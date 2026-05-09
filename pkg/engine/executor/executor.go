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
	testPath := resolvePath(opts.BasePath, rule.Test)
	ev := baseEvent(rule, opts, "dry-run")

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
	testPath := resolvePath(opts.BasePath, rule.Test)
	actionPath := resolvePath(opts.BasePath, rule.Action)
	ev := baseEvent(rule, opts, "apply")

	// 1. Test : skip si déjà conforme.
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

	// 2. Action : lance .action.ps1.
	actionStart := time.Now()
	actionOut, _, actionErr := runPS(ctx, opts.Runner, actionPath, nil, timeout)
	totalDur := time.Since(actionStart) + testDur
	ev["duration_ms"] = totalDur.Milliseconds()

	if actionErr == nil {
		ev["status"] = "applied"
		ev["before"] = actionOut["before"]
		ev["after"] = actionOut["after"]
		sum.Applied++
		return ev, false
	}

	// 3. Action a planté → tenter auto-rollback.
	ev["status"] = "failed"
	ev["error"] = fmt.Sprintf("action.ps1: %v", actionErr)

	// Le before pour le rollback : on n'a pas eu le retour de l'action (qui aurait
	// contenu before). On peut pas restaurer l'état. On émet juste un rollback_result
	// disant qu'on n'a pas pu rollback.
	if rule.Undo == "" || rule.Irreversible {
		ev["rollback"] = "skipped (rule is irreversible or has no undo)"
		sum.Failed++
		return ev, true
	}

	// On utilise le testOut.current comme proxy pour le before (l'état avant
	// l'action est ce que test a vu juste avant de lancer l'action).
	rollbackInput := testOut["current"]
	undoPath := resolvePath(opts.BasePath, rule.Undo)
	_, undoDur, undoErr := runPS(ctx, opts.Runner, undoPath, rollbackInput, timeout)

	rollbackEv := map[string]any{
		"type":         "rollback_result",
		"run_id":       opts.RunID,
		"section_id":   ev["section_id"],
		"rule_id":      rule.ID,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"duration_ms":  undoDur.Milliseconds(),
		"trigger":      "action_failed",
		"trigger_err":  actionErr.Error(),
	}
	if undoErr != nil {
		rollbackEv["status"] = "rollback_failed"
		rollbackEv["error"] = undoErr.Error()
		ev["rollback"] = "failed"
		sum.Failed++
	} else {
		rollbackEv["status"] = "rollback_ok"
		ev["rollback"] = "ok"
		ev["status"] = "rolled_back"
		sum.RolledBack++
	}
	_ = opts.Writer.Emit(rollbackEv)

	return ev, true // abort le run global
}

func baseEvent(rule manifest.Rule, opts Options, mode string) map[string]any {
	ev := map[string]any{
		"type":      "action_result",
		"run_id":    opts.RunID,
		"section_id": ruleSectionID(rule),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"rule_id":   rule.ID,
		"mode":      mode,
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

// resolvePath retourne path tel quel s'il est absolu (cas où le manifest a
// utilisé un chemin absolu, p.ex. dans des tests), sinon joint avec base.
//
// Sur Windows, filepath.Join("C:\\base", "C:\\foo") retourne "C:\\base\\C:\\foo"
// (Go ne drop pas le second drive letter), donc on doit faire le check à la main.
func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
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
