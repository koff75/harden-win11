// Package executor exécute les règles d'un manifest section, soit en mode
// dry-run (lance .test.ps1 et reporte would_skip/would_apply/would_fail),
// soit en mode apply (lance .action.ps1, capture before/after, et déclenche
// un auto-rollback via .undo.ps1 si l'action plante).
package executor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// DefaultRuleTimeout est le timeout par défaut appliqué à chaque
// invocation de .test.ps1 / .action.ps1 si Options.RuleTimeout est zéro.
const DefaultRuleTimeout = 30 * time.Second

// Mode contrôle si Run() exécute en lecture seule (Dry) ou en mode réel (Apply).
type Mode int

const (
	// ModeDry n'exécute que .test.ps1 et reporte would_skip/would_apply/would_fail.
	// Aucune modification système.
	ModeDry Mode = iota
	// ModeApply lance .test.ps1 d'abord (skip si déjà conforme) puis .action.ps1
	// si non-conforme. Si .action.ps1 plante après avoir capturé before, déclenche
	// .undo.ps1 (auto-rollback) puis stoppe l'exécution.
	ModeApply
)

// Options configure une exécution.
type Options struct {
	Mode        Mode
	ManifestDir string
	BasePath    string
	Runner      *runner.Runner
	Writer      *ndjson.Writer
	RunID       string
	// RuleTimeout limite la durée d'exécution de chaque .test.ps1 / .action.ps1.
	// Zéro = DefaultRuleTimeout (30s).
	RuleTimeout time.Duration
	// Profile filtre les rules sur leur champ rule.profiles. Vide = pas de
	// filtre (toutes les rules tournent). Sinon : seules les rules dont
	// rule.profiles contient cette valeur (ou rule.profiles vide) tournent.
	Profile string
	// Severity filtre par rule.severity (critical | important | nice-to-have).
	// Vide = toutes les severities. Permet à l'utilisateur de faire des vagues
	// manuelles : `--severity critical` puis reboot puis `--severity important`.
	Severity string
	// ExcludedRuleIDs : rule_ids à skipper explicitement (cas où l'utilisateur
	// décoche manuellement des règles dans la GUI). Appliqué après le filtre
	// profile.
	ExcludedRuleIDs map[string]bool
	// Parallel : nombre max de rules dry-run exécutées en parallèle (default 1).
	// N'a d'effet qu'en ModeDry — apply reste séquentiel à cause de l'auto-rollback.
	// Les events sont buffer-réordonnés pour rester dans l'ordre du manifest.
	Parallel int
}

// Summary agrège les compteurs de statuts d'un Run.
//
// En ModeDry  : Skipped = would_skip, Applied = would_apply, Failed = would_fail.
// En ModeApply : Skipped = skipped (déjà conforme), Applied = applied (action OK),
// Failed = failed (action plantée, pas de rollback ou rollback échoué),
// RolledBack = rolled_back (action plantée + rollback OK).
type Summary struct {
	Skipped    int
	Applied    int
	Failed     int
	RolledBack int
}

// ErrAborted est retourné par Run quand l'apply s'est arrêté à mi-chemin
// (typiquement après un auto-rollback). Le caller doit propager pour que
// la CLI émette un exit code non-zero.
var ErrAborted = errors.New("apply aborted")

// Run exécute les règles d'un manifest section selon le mode.
//
// Émet un event "section_start" puis une suite "action_result" puis
// "section_end". Les events "run_start" et "run_end" englobants sont
// émis par le caller (CLI), pas ici.
//
// En ModeApply, si une action plante, déclenche le rollback via .undo.ps1
// (avec le before capturé), émet "rollback_result", puis retourne ErrAborted
// pour stopper le run global.
func Run(ctx context.Context, sectionPath string, opts Options) (Summary, error) {
	var sum Summary
	s, err := manifest.Load(sectionPath)
	if err != nil {
		return sum, fmt.Errorf("load manifest: %w", err)
	}

	// Filtrer les rules selon le profil sélectionné.
	rules := s.Rules
	if opts.Profile != "" {
		filtered := make([]manifest.Rule, 0, len(rules))
		for _, r := range rules {
			if r.AppliesToProfile(opts.Profile) {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}
	// Filtre par severity (vagues progressives : critical → important → nice-to-have).
	if opts.Severity != "" {
		filtered := make([]manifest.Rule, 0, len(rules))
		for _, r := range rules {
			if r.Severity == opts.Severity {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}
	// Filtre supplémentaire : rule_ids exclus explicitement par l'utilisateur.
	if len(opts.ExcludedRuleIDs) > 0 {
		filtered := make([]manifest.Rule, 0, len(rules))
		for _, r := range rules {
			if !opts.ExcludedRuleIDs[r.ID] {
				filtered = append(filtered, r)
			}
		}
		rules = filtered
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":             "section_start",
		"run_id":           opts.RunID,
		"section_id":       s.Section.ID,
		"section_order":    s.Section.Order,
		"section_title":    s.Section.Title,
		"manifest_version": s.Version,
		"rule_count":       len(rules),
		"mode":             modeName(opts.Mode),
		"profile":          opts.Profile,
	})

	timeout := opts.RuleTimeout
	if timeout == 0 {
		timeout = DefaultRuleTimeout
	}

	// Apply reste séquentiel (auto-rollback critique). Dry-run peut être parallélisé.
	if opts.Mode == ModeDry && opts.Parallel > 1 && len(rules) > 1 {
		runDryParallel(ctx, rules, opts, timeout, &sum)
	} else {
		for _, rule := range rules {
			var (
				ev      map[string]any
				aborted bool
			)
			switch opts.Mode {
			case ModeDry:
				ev = runDry(ctx, rule, opts, timeout, &sum)
			case ModeApply:
				ev, aborted = runApply(ctx, rule, opts, timeout, &sum)
			default:
				ev = map[string]any{
					"type":    "action_result",
					"run_id":  opts.RunID,
					"rule_id": rule.ID,
					"status":  "failed",
					"error":   fmt.Sprintf("unknown executor mode: %d", opts.Mode),
				}
				sum.Failed++
			}
			_ = opts.Writer.Emit(ev)

			if aborted {
				_ = opts.Writer.Emit(map[string]any{
					"type":       "section_end",
					"run_id":     opts.RunID,
					"section_id": s.Section.ID,
					"aborted":    true,
				})
				return sum, ErrAborted
			}
		}
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":       "section_end",
		"run_id":     opts.RunID,
		"section_id": s.Section.ID,
	})
	return sum, nil
}

func modeName(m Mode) string {
	switch m {
	case ModeDry:
		return "dry-run"
	case ModeApply:
		return "apply"
	default:
		return "unknown"
	}
}

// runDryParallel : N workers consomment les rules, leurs résultats sont
// rebuffered dans l'ordre original avant émission, pour préserver l'ordre
// utile en debug (et l'UX si la GUI passe Parallel > 1, ce qu'elle ne fait
// pas pour l'instant).
//
// Apply ne passe jamais ici (l'auto-rollback nécessite un ordre strict
// pour décider d'aborter la section).
func runDryParallel(ctx context.Context, rules []manifest.Rule, opts Options, timeout time.Duration, sum *Summary) {
	n := opts.Parallel
	if n > len(rules) {
		n = len(rules)
	}

	type job struct {
		idx  int
		rule manifest.Rule
	}
	jobs := make(chan job, n*2)
	results := make([]map[string]any, len(rules))
	deltas := make([]Summary, len(rules))

	var wg sync.WaitGroup
	for w := 0; w < n; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				var local Summary
				ev := runDry(ctx, j.rule, opts, timeout, &local)
				results[j.idx] = ev
				deltas[j.idx] = local
			}
		}()
	}
	for i, r := range rules {
		jobs <- job{idx: i, rule: r}
	}
	close(jobs)
	wg.Wait()

	// Émission ordonnée + sommage des deltas.
	for i, ev := range results {
		_ = opts.Writer.Emit(ev)
		sum.Skipped += deltas[i].Skipped
		sum.Applied += deltas[i].Applied
		sum.Failed += deltas[i].Failed
	}
}

// runDry exécute la logique dry-run pour une règle.
func runDry(ctx context.Context, rule manifest.Rule, opts Options, timeout time.Duration, sum *Summary) map[string]any {
	ev := baseEvent(rule, opts, "dry-run")
	testPath, err := resolvePath(opts.BasePath, rule.Test)
	if err != nil {
		ev["status"] = "would_fail"
		ev["error"] = fmt.Sprintf("resolve test path: %v", err)
		sum.Failed++
		return ev
	}

	out, dur, err := runPS(ctx, opts.Runner, testPath, nil, timeout)
	ev["duration_ms"] = dur.Milliseconds()

	if err != nil {
		ev["status"] = "would_fail"
		ev["error"] = err.Error()
		sum.Failed++
		return ev
	}

	compliant, status, statusErr := evalCompliant(out)
	if statusErr != nil {
		ev["status"] = "would_fail"
		ev["error"] = statusErr.Error()
		sum.Failed++
		return ev
	}
	if compliant {
		ev["status"] = "would_skip"
		ev["reason"] = "already_compliant"
		ev["current_state"] = out["current"]
		sum.Skipped++
	} else {
		ev["status"] = "would_apply"
		ev["current_state"] = out["current"]
		sum.Applied++
	}
	_ = status
	return ev
}

// runApply exécute la logique apply pour une règle. Retourne (event, aborted).
// aborted=true signale au caller de stopper le run global (auto-rollback enclenché).
func runApply(ctx context.Context, rule manifest.Rule, opts Options, timeout time.Duration, sum *Summary) (map[string]any, bool) {
	ev := baseEvent(rule, opts, "apply")
	testPath, err := resolvePath(opts.BasePath, rule.Test)
	if err != nil {
		ev["status"] = "failed"
		ev["error"] = fmt.Sprintf("resolve test path: %v", err)
		sum.Failed++
		return ev, false
	}
	actionPath, err := resolvePath(opts.BasePath, rule.Action)
	if err != nil {
		ev["status"] = "failed"
		ev["error"] = fmt.Sprintf("resolve action path: %v", err)
		sum.Failed++
		return ev, false
	}

	// 1. Test : skip si déjà conforme OU si la feature est en cours d'utilisation.
	testOut, testDur, testErr := runPS(ctx, opts.Runner, testPath, nil, timeout)
	if testErr != nil {
		ev["duration_ms"] = testDur.Milliseconds()
		ev["status"] = "failed"
		ev["error"] = fmt.Sprintf("test.ps1: %v", testErr)
		sum.Failed++
		return ev, false
	}
	compliant, _, statusErr := evalCompliant(testOut)
	if statusErr != nil {
		ev["duration_ms"] = testDur.Milliseconds()
		ev["status"] = "failed"
		ev["error"] = statusErr.Error()
		sum.Failed++
		return ev, false
	}
	if compliant {
		ev["status"] = "skipped"
		ev["reason"] = "already_compliant"
		ev["current_state"] = testOut["current"]
		ev["duration_ms"] = testDur.Milliseconds()
		sum.Skipped++
		return ev, false
	}
	// Détection "feature en cours d'utilisation" : si .test.ps1 retourne
	// feature_in_use=true (ex: session RDP active, partage SMB legacy connecté,
	// proxy WPAD utilisé), on refuse de casser à chaud — mieux vaut un skip
	// qu'une coupure brutale. L'utilisateur doit manuellement fermer la session
	// et réessayer.
	if inUse, ok := testOut["feature_in_use"].(bool); ok && inUse {
		ev["status"] = "skipped"
		ev["reason"] = "feature_in_use"
		ev["current_state"] = testOut["current"]
		ev["duration_ms"] = testDur.Milliseconds()
		if msg, ok := testOut["feature_in_use_reason"].(string); ok {
			ev["feature_in_use_reason"] = msg
		}
		sum.Skipped++
		return ev, false
	}

	// 2. Action : lance .action.ps1.
	actionStart := time.Now()
	actionOut, _, actionErr := runPS(ctx, opts.Runner, actionPath, nil, timeout)
	totalDur := time.Since(actionStart) + testDur
	ev["duration_ms"] = totalDur.Milliseconds()

	// Si l'action a retourne explicitement ok=false dans son JSON (sans crash
	// PS), on traite comme un echec metier propre — l'action a detecte qu'elle
	// ne pouvait pas faire son boulot (ex: feature non supportee sur cette
	// edition Windows, dependance manuelle requise). Pas de post-apply test
	// dans ce cas (l'action n'a rien modifie), pas non plus de rollback.
	if actionErr == nil {
		if okVal, hasOk := actionOut["ok"]; hasOk {
			if okBool, isBool := okVal.(bool); isBool && !okBool {
				ev["status"] = "failed"
				if msg, ok := actionOut["error"].(string); ok && msg != "" {
					ev["error"] = msg
				} else {
					ev["error"] = "action returned ok=false without error message"
				}
				if before, ok := actionOut["before"]; ok {
					ev["before"] = before
				}
				if after, ok := actionOut["after"]; ok {
					ev["after"] = after
				}
				ev["rollback"] = "not_attempted (action did not modify system)"
				sum.Failed++
				return ev, false // continue avec les regles suivantes
			}
		}
	}

	// Re-test post-apply : double barrière indépendante. Une action qui retourne
	// ok=true peut quand même n'avoir rien changé si une GPO ré-écrase, ou si la
	// règle ment. On relance .test.ps1 ; si non-conforme malgré ok=true → on
	// considère l'apply comme failed et on déclenche un rollback (best-effort).
	if actionErr == nil {
		recheckOut, _, recheckErr := runPS(ctx, opts.Runner, testPath, nil, timeout)
		if recheckErr != nil {
			// Re-test a planté (cas rare). On garde "applied" mais on log la raison.
			ev["status"] = "applied"
			ev["before"] = actionOut["before"]
			ev["after"] = actionOut["after"]
			ev["recheck"] = "test_failed"
			ev["recheck_error"] = recheckErr.Error()
			sum.Applied++
			return ev, false
		}
		recheckCompliant, _, _ := evalCompliant(recheckOut)
		if recheckCompliant {
			ev["status"] = "applied"
			ev["before"] = actionOut["before"]
			ev["after"] = actionOut["after"]
			ev["recheck"] = "compliant"
			sum.Applied++
			return ev, false
		}
		// Action ok mais re-test non-conforme → action menteuse / GPO re-écrase / etc.
		// On bascule en mode "failed" et on tombe dans le path rollback ci-dessous.
		ev["error"] = "action.ps1 returned ok=true but post-apply re-test reports non-compliant (GPO override or silent failure)"
		ev["recheck"] = "non_compliant"
		ev["recheck_state"] = recheckOut["current"]
	} else {
		// 3. Action a vraiment planté → fall-through vers rollback.
		ev["error"] = fmt.Sprintf("action.ps1: %v", actionErr)
	}
	ev["status"] = "failed"

	// Tenter auto-rollback. Best-effort : si la rule est irreversible ou n'a pas
	// d'undo, on ne peut rien faire. Mais on CONTINUE avec les regles suivantes
	// plutot que d'aborter tout le run — le rollback "skipped" signifie juste
	// qu'on n'a pas pu restaurer cette rule specifique, pas que le systeme est
	// dans un etat instable. Coherent avec la philosophie "rollback ok = safe
	// to continue" appliquee plus bas.
	if rule.Undo == "" || rule.Irreversible {
		ev["rollback"] = "skipped (rule is irreversible or has no undo)"
		sum.Failed++
		return ev, false
	}

	// Le before pour le rollback : on prend en priorité actionOut.before (capturé
	// par .action.ps1 juste avant la modif), sinon testOut.current (état observé
	// par .test.ps1 avant l'apply).
	var rollbackInput any
	if actionOut != nil {
		if before, ok := actionOut["before"]; ok && before != nil {
			rollbackInput = before
		}
	}
	if rollbackInput == nil {
		rollbackInput = testOut["current"]
	}
	undoPath, undoResolveErr := resolvePath(opts.BasePath, rule.Undo)
	if undoResolveErr != nil {
		// Si on ne peut même pas résoudre le path d'undo, on est en état système
		// inconnu (action a peut-être modifié des trucs) — on aborte tout le run.
		rollbackEv := map[string]any{
			"type":        "rollback_result",
			"run_id":      opts.RunID,
			"section_id":  ev["section_id"],
			"rule_id":     rule.ID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"status":      "rollback_failed",
			"error":       fmt.Sprintf("resolve undo path: %v", undoResolveErr),
			"trigger":     "rollback_path_invalid",
		}
		ev["rollback"] = "failed"
		sum.Failed++
		_ = opts.Writer.Emit(rollbackEv)
		return ev, true
	}
	_, undoDur, undoErr := runPS(ctx, opts.Runner, undoPath, rollbackInput, timeout)

	// actionErr peut être nil si on est arrivé ici via le path "recheck_failed"
	// (action ok=true mais le re-test post-apply dit non-conforme).
	triggerErr := ""
	trigger := "action_failed"
	if actionErr != nil {
		triggerErr = actionErr.Error()
	} else {
		trigger = "recheck_failed"
		if errStr, ok := ev["error"].(string); ok {
			triggerErr = errStr
		}
	}
	rollbackEv := map[string]any{
		"type":        "rollback_result",
		"run_id":      opts.RunID,
		"section_id":  ev["section_id"],
		"rule_id":     rule.ID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"duration_ms": undoDur.Milliseconds(),
		"trigger":     trigger,
		"trigger_err": triggerErr,
	}
	if undoErr != nil {
		rollbackEv["status"] = "rollback_failed"
		rollbackEv["error"] = undoErr.Error()
		ev["rollback"] = "failed"
		sum.Failed++
		_ = opts.Writer.Emit(rollbackEv)
		// Rollback failed → état système inconnu, on aborte pour eviter
		// de continuer à modifier un système instable.
		return ev, true
	}

	rollbackEv["status"] = "rollback_ok"
	ev["rollback"] = "ok"
	ev["status"] = "rolled_back"
	sum.RolledBack++
	_ = opts.Writer.Emit(rollbackEv)

	// Rollback réussi → état pré-apply restauré, safe de continuer avec les
	// règles suivantes. L'user verra X lignes "Modif annulée" + tout le reste
	// appliqué — plus utile que tout stopper sur la première (qui peut etre
	// symptomatique d'un Tamper Protection, GPO, etc. qui affecte tout un
	// pan, mais pas le reste).
	return ev, false
}

func baseEvent(rule manifest.Rule, opts Options, mode string) map[string]any {
	ev := map[string]any{
		"type":       "action_result",
		"run_id":     opts.RunID,
		"section_id": ruleSectionID(rule),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"rule_id":    rule.ID,
		"mode":       mode,
	}
	if mode == "dry-run" {
		ev["dry_run"] = true
	} else {
		ev["dry_run"] = false
	}
	return ev
}

// ruleSectionID retourne la première composante du rule.id (avant le '.').
// Utile pour étiqueter les events sans avoir à passer s.Section.ID partout.
func ruleSectionID(rule manifest.Rule) string {
	for i, c := range rule.ID {
		if c == '.' {
			return rule.ID[:i]
		}
	}
	return rule.ID
}

// resolvePath joint path à base et garantit que le résultat reste contenu
// sous base. Refuse les chemins absolus et les traversées (../).
//
// Why: un manifest YAML est attacker-controlled dès qu'il peut être planté
// dans un dossier que le binaire élevé consomme (cf. findRepoLayout). Sans
// containment check, un manifest peut référencer C:\Windows\System32\evil.ps1
// ou ../../../../evil.ps1, et le runner l'exécute avec les privilèges du
// process (typiquement admin).
func resolvePath(base, path string) (string, error) {
	if path == "" {
		return "", errors.New("resolvePath: path is empty")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("resolvePath: absolute paths are rejected (got %q)", path)
	}
	// Sur Windows, refuser aussi les paths UNC type \\server\share qui ne sont
	// pas IsAbs() techniquement mais sortent du containment.
	if strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//") {
		return "", fmt.Errorf("resolvePath: UNC paths are rejected (got %q)", path)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolvePath: invalid base %q: %w", base, err)
	}
	joined := filepath.Join(absBase, path)
	rel, err := filepath.Rel(absBase, joined)
	if err != nil {
		return "", fmt.Errorf("resolvePath: cannot compute relative path: %w", err)
	}
	// rel commence par ".." si joined sort de absBase.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolvePath: path %q escapes base %q (resolved to %q)", path, base, joined)
	}
	return joined, nil
}

func runPS(ctx context.Context, r *runner.Runner, path string, input any, timeout time.Duration) (map[string]any, time.Duration, error) {
	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	out, err := r.RunPS(rctx, path, input)
	return out, time.Since(start), err
}

// evalCompliant lit le champ 'compliant' du JSON retourné par .test.ps1 et
// retourne (compliant, statusName, error).
//
// Si le champ est manquant ou pas un bool, retourne une erreur explicite.
func evalCompliant(out map[string]any) (bool, string, error) {
	raw, ok := out["compliant"]
	if !ok {
		return false, "", errors.New("test.ps1 output missing required 'compliant' field")
	}
	b, isBool := raw.(bool)
	if !isBool {
		return false, "", fmt.Errorf("test.ps1 'compliant' field must be a boolean (got %T: %v)", raw, raw)
	}
	return b, "", nil
}
