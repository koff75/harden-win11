// harden-engine est le moteur CLI v2 du projet harden-win11.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/executor"
	"github.com/koff75/harden-win11/pkg/engine/journal"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
	"github.com/koff75/harden-win11/pkg/engine/winadmin"
	"github.com/spf13/cobra"
)

var (
	Version         = "0.1.0-dev"
	ManifestVersion = "1.0"
)

var (
	flagManifestDir string
	flagSchemaPath  string
	flagDryRun      bool
	flagSection     string
	flagRuleTimeout time.Duration
	flagJournalDir  string
	flagYes         bool
	flagRunID       string
	flagRuleID      string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Moteur de hardening Windows 11",
		Long:  "harden-engine — moteur de la baseline de sécurité Windows 11 v2.",
	}
	rootCmd.PersistentFlags().StringVar(&flagManifestDir, "manifest-dir", "manifests", "Dossier contenant les manifests YAML")
	rootCmd.PersistentFlags().StringVar(&flagSchemaPath, "schema", "schemas/manifest.schema.json", "Chemin du JSONSchema")
	rootCmd.PersistentFlags().StringVar(&flagJournalDir, "journal-dir", "", "Dossier du journal NDJSON (vide = %ProgramData%\\Harden-Win11\\runs\\)")

	rootCmd.AddCommand(versionCmd(), validateCmd(), applyCmd(), undoCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Affiche la version (engine + manifest + OS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := map[string]any{
				"version":          Version,
				"manifest_version": ManifestVersion,
				"go":               runtime.Version(),
				"os":               runtime.GOOS,
				"arch":             runtime.GOARCH,
			}
			b, err := json.Marshal(out)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
			return nil
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Valide tous les manifests contre le JSONSchema",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, err := loadAndValidateManifests(flagManifestDir, flagSchemaPath, true /*verbose*/)
			return err
		},
	}
}

// manifestEntry décrit un manifest valide chargé depuis le dossier.
type manifestEntry struct {
	path  string
	order int
	id    string
}

// loadAndValidateManifests parse + valide tous les manifests YAML d'un dossier.
// Si verbose, écrit [OK]/[FAIL]/[COLLISION] sur stderr (cas validateCmd).
// Retourne la liste des entries valides (path + section meta).
func loadAndValidateManifests(dir, schemaPath string, verbose bool) ([]manifestEntry, int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("read manifest dir: %w", err)
	}

	validator, err := manifest.NewValidator(schemaPath)
	if err != nil {
		return nil, 0, &exitError{code: 4, msg: err.Error()}
	}

	var (
		failed     int
		processed  int
		validPaths []string
	)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		processed++
		path := filepath.Join(dir, e.Name())
		if err := validator.ValidateFile(path); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[FAIL] %s : %v\n", e.Name(), err)
			}
			failed++
		} else {
			if verbose {
				fmt.Fprintf(os.Stderr, "[OK]   %s\n", e.Name())
			}
			validPaths = append(validPaths, path)
		}
	}

	if processed == 0 {
		return nil, 0, &exitError{code: 4, msg: fmt.Sprintf("no manifests found in %s (expected *.yaml or *.yml)", dir)}
	}

	// Collision detection sur section.id.
	seen := map[string]string{}
	var collisions int
	var sections []manifestEntry
	for _, p := range validPaths {
		s, err := manifest.Load(p)
		if err != nil {
			continue
		}
		if existing, ok := seen[s.Section.ID]; ok {
			if verbose {
				fmt.Fprintf(os.Stderr, "[COLLISION] section.id %q is defined in both %s and %s\n",
					s.Section.ID, filepath.Base(existing), filepath.Base(p))
			}
			collisions++
		} else {
			seen[s.Section.ID] = p
			sections = append(sections, manifestEntry{path: p, order: s.Section.Order, id: s.Section.ID})
		}
	}

	if failed > 0 {
		return nil, failed, &exitError{code: 3, msg: fmt.Sprintf("%d manifests invalid", failed)}
	}
	if collisions > 0 {
		return nil, collisions, &exitError{code: 3, msg: fmt.Sprintf("%d section.id collisions", collisions)}
	}
	return sections, 0, nil
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Exécute (réel ou dry-run) les règles. Sans --section = toutes les sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagRuleTimeout < 0 {
				return &exitError{code: 4, msg: "--rule-timeout must be >= 0 (use 0 to keep the default 30s)"}
			}

			// Re-validation TOUJOURS avant apply (sécurité : trust + verify).
			allSections, _, err := loadAndValidateManifests(flagManifestDir, flagSchemaPath, false)
			if err != nil {
				return err
			}

			// Filtrer sur --section si demandé.
			var sections []manifestEntry
			for _, s := range allSections {
				if flagSection != "" && s.id != flagSection {
					continue
				}
				sections = append(sections, s)
			}
			if len(sections) == 0 {
				if flagSection != "" {
					return &exitError{code: 4, msg: fmt.Sprintf("section %q not found in %s", flagSection, flagManifestDir)}
				}
				return &exitError{code: 4, msg: fmt.Sprintf("no manifests found in %s (expected *.yaml)", flagManifestDir)}
			}

			sort.Slice(sections, func(i, j int) bool {
				if sections[i].order != sections[j].order {
					return sections[i].order < sections[j].order
				}
				return sections[i].id < sections[j].id
			})

			mode := executor.ModeDry
			if !flagDryRun {
				mode = executor.ModeApply
			}

			// Mode apply réel : check admin + confirmation user.
			if mode == executor.ModeApply {
				isAdmin, err := winadmin.IsElevated()
				if err != nil {
					return &exitError{code: 5, msg: fmt.Sprintf("admin check failed: %v", err)}
				}
				if !isAdmin {
					return &exitError{code: 5, msg: "apply (without --dry-run) requires Administrator privileges. Re-run from an elevated PowerShell or use --dry-run."}
				}
				if !flagYes {
					fmt.Fprintf(os.Stderr, "About to apply %d section(s) :\n", len(sections))
					for _, s := range sections {
						fmt.Fprintf(os.Stderr, "  - %s\n", s.id)
					}
					fmt.Fprint(os.Stderr, "Type 'yes' to confirm (or use --yes to skip): ")
					var ans string
					_, _ = fmt.Scanln(&ans)
					if strings.ToLower(strings.TrimSpace(ans)) != "yes" {
						return &exitError{code: 6, msg: "apply cancelled by user"}
					}
				}
			}

			absManifestDir, _ := filepath.Abs(flagManifestDir)
			base := filepath.Dir(absManifestDir)

			runID := time.Now().UTC().Format("2006-01-02T15-04-05")

			// Le journal disque + stdout (sauf en mode dry-run où le disque est optionnel).
			outputs, journalPath, err := openOutputs(runID, mode == executor.ModeApply)
			if err != nil {
				return &exitError{code: 4, msg: fmt.Sprintf("open journal: %v", err)}
			}
			defer outputs.Close()
			w := ndjson.NewWriter(outputs)
			ctx := context.Background()

			sectionIDs := make([]string, 0, len(sections))
			for _, s := range sections {
				sectionIDs = append(sectionIDs, s.id)
			}
			_ = w.Emit(map[string]any{
				"type":             "run_start",
				"run_id":           runID,
				"manifest_version": ManifestVersion,
				"engine_version":   Version,
				"mode":             modeName(mode),
				"dry_run":          mode == executor.ModeDry,
				"section_count":    len(sections),
				"sections":         sectionIDs,
				"journal_path":     journalPath,
			})

			var total executor.Summary
			var aborted bool
			var abortedSection string
			for _, sct := range sections {
				summary, err := executor.Run(ctx, sct.path, executor.Options{
					Mode:        mode,
					ManifestDir: flagManifestDir,
					BasePath:    base,
					Runner:      runner.New(),
					Writer:      w,
					RunID:       runID,
					RuleTimeout: flagRuleTimeout,
				})
				total.Skipped += summary.Skipped
				total.Applied += summary.Applied
				total.Failed += summary.Failed
				total.RolledBack += summary.RolledBack

				if errors.Is(err, executor.ErrAborted) {
					aborted = true
					abortedSection = sct.id
					break
				}
				if err != nil {
					_ = w.Emit(map[string]any{
						"type":   "run_end",
						"run_id": runID,
						"error":  err.Error(),
					})
					return fmt.Errorf("section %s: %w", sct.id, err)
				}
			}

			runEnd := map[string]any{
				"type":        "run_end",
				"run_id":      runID,
				"skipped":     total.Skipped,
				"applied":     total.Applied,
				"failed":      total.Failed,
				"rolled_back": total.RolledBack,
			}
			if aborted {
				runEnd["aborted"] = true
				runEnd["aborted_section"] = abortedSection
			}
			_ = w.Emit(runEnd)

			if aborted {
				return &exitError{code: 2, msg: fmt.Sprintf("apply aborted in section %q after auto-rollback", abortedSection)}
			}
			if total.Failed > 0 {
				return &exitError{code: 2, msg: fmt.Sprintf("%d rules failed (status=failed)", total.Failed)}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Mode dry-run (lit l'état, ne modifie rien)")
	cmd.Flags().StringVar(&flagSection, "section", "", "ID de la section à exécuter (vide = toutes)")
	cmd.Flags().DurationVar(&flagRuleTimeout, "rule-timeout", executor.DefaultRuleTimeout, "Timeout maximum par règle (ex: 30s, 1m)")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Skip la confirmation interactive avant apply réel")
	return cmd
}

func undoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Restaure l'état avant un run via les .undo.ps1 (lit le journal NDJSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagRuleTimeout < 0 {
				return &exitError{code: 4, msg: "--rule-timeout must be >= 0"}
			}

			journalDir := flagJournalDir
			if journalDir == "" {
				journalDir = journal.DefaultDir()
			}

			runIDToUse := flagRunID
			if runIDToUse == "" {
				latest, err := journal.LatestRunID(journalDir)
				if err != nil {
					return &exitError{code: 4, msg: fmt.Sprintf("find latest run: %v", err)}
				}
				runIDToUse = latest
			}

			events, err := journal.ReadRun(journalDir, runIDToUse)
			if err != nil {
				return &exitError{code: 4, msg: fmt.Sprintf("read journal: %v", err)}
			}

			// On ne peut undo que les règles avec status=applied (pas would_apply, pas
			// skipped, pas failed).
			var toUndo []journal.AppliedRule
			for _, ev := range events {
				if ev["type"] != "action_result" {
					continue
				}
				if ev["status"] != "applied" {
					continue
				}
				ruleID, _ := ev["rule_id"].(string)
				sectionID, _ := ev["section_id"].(string)
				if flagRuleID != "" && ruleID != flagRuleID {
					continue
				}
				toUndo = append(toUndo, journal.AppliedRule{
					RuleID:    ruleID,
					SectionID: sectionID,
					Before:    ev["before"],
				})
			}

			if len(toUndo) == 0 {
				if flagRuleID != "" {
					return &exitError{code: 4, msg: fmt.Sprintf("no applied rule %q found in run %s", flagRuleID, runIDToUse)}
				}
				return &exitError{code: 4, msg: fmt.Sprintf("no applied rules found in run %s", runIDToUse)}
			}

			// Admin requis pour les .undo.ps1 qui touchent au système.
			isAdmin, err := winadmin.IsElevated()
			if err != nil {
				return &exitError{code: 5, msg: fmt.Sprintf("admin check failed: %v", err)}
			}
			if !isAdmin {
				return &exitError{code: 5, msg: "undo requires Administrator privileges. Re-run from an elevated PowerShell."}
			}

			if !flagYes {
				fmt.Fprintf(os.Stderr, "About to undo %d rule(s) from run %s :\n", len(toUndo), runIDToUse)
				for _, u := range toUndo {
					fmt.Fprintf(os.Stderr, "  - %s\n", u.RuleID)
				}
				fmt.Fprint(os.Stderr, "Type 'yes' to confirm (or use --yes to skip): ")
				var ans string
				_, _ = fmt.Scanln(&ans)
				if strings.ToLower(strings.TrimSpace(ans)) != "yes" {
					return &exitError{code: 6, msg: "undo cancelled by user"}
				}
			}

			// Charger les manifests pour retrouver les paths .undo.ps1.
			allSections, _, err := loadAndValidateManifests(flagManifestDir, flagSchemaPath, false)
			if err != nil {
				return err
			}
			rulesByID := map[string]ruleRef{}
			for _, e := range allSections {
				s, err := manifest.Load(e.path)
				if err != nil {
					continue
				}
				for _, r := range s.Rules {
					rulesByID[r.ID] = ruleRef{undo: r.Undo, irreversible: r.Irreversible}
				}
			}

			absManifestDir, _ := filepath.Abs(flagManifestDir)
			base := filepath.Dir(absManifestDir)
			undoRunID := time.Now().UTC().Format("2006-01-02T15-04-05")

			outputs, undoJournalPath, err := openOutputs("undo-"+undoRunID, true)
			if err != nil {
				return &exitError{code: 4, msg: fmt.Sprintf("open journal: %v", err)}
			}
			defer outputs.Close()
			w := ndjson.NewWriter(outputs)

			_ = w.Emit(map[string]any{
				"type":           "run_start",
				"run_id":         "undo-" + undoRunID,
				"engine_version": Version,
				"mode":           "undo",
				"target_run_id":  runIDToUse,
				"rule_count":     len(toUndo),
				"journal_path":   undoJournalPath,
			})

			r := runner.New()
			ctx := context.Background()
			timeout := flagRuleTimeout
			if timeout == 0 {
				timeout = executor.DefaultRuleTimeout
			}

			var ok, failed int
			// Inverse l'ordre : LIFO.
			for i := len(toUndo) - 1; i >= 0; i-- {
				u := toUndo[i]
				ref, found := rulesByID[u.RuleID]
				if !found {
					_ = w.Emit(map[string]any{
						"type":    "undo_result",
						"run_id":  "undo-" + undoRunID,
						"rule_id": u.RuleID,
						"status":  "skipped",
						"reason":  "rule not found in current manifests",
					})
					continue
				}
				if ref.undo == "" || ref.irreversible {
					_ = w.Emit(map[string]any{
						"type":    "undo_result",
						"run_id":  "undo-" + undoRunID,
						"rule_id": u.RuleID,
						"status":  "skipped",
						"reason":  "rule is irreversible or has no undo",
					})
					continue
				}
				undoPath := filepath.Join(base, ref.undo)
				ctxRule, cancel := context.WithTimeout(ctx, timeout)
				start := time.Now()
				_, err := r.RunPS(ctxRule, undoPath, u.Before)
				dur := time.Since(start)
				cancel()
				ev := map[string]any{
					"type":        "undo_result",
					"run_id":      "undo-" + undoRunID,
					"rule_id":     u.RuleID,
					"section_id":  u.SectionID,
					"timestamp":   time.Now().UTC().Format(time.RFC3339),
					"duration_ms": dur.Milliseconds(),
				}
				if err != nil {
					ev["status"] = "failed"
					ev["error"] = err.Error()
					failed++
				} else {
					ev["status"] = "ok"
					ok++
				}
				_ = w.Emit(ev)
			}

			_ = w.Emit(map[string]any{
				"type":    "run_end",
				"run_id":  "undo-" + undoRunID,
				"ok":      ok,
				"failed":  failed,
			})

			if failed > 0 {
				return &exitError{code: 2, msg: fmt.Sprintf("%d undo operations failed", failed)}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagRunID, "run-id", "", "Run ID dont undo les règles (vide = dernier run)")
	cmd.Flags().StringVar(&flagRuleID, "rule-id", "", "Cibler une règle précise (vide = toutes les règles applied du run)")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Skip la confirmation interactive")
	cmd.Flags().DurationVar(&flagRuleTimeout, "rule-timeout", executor.DefaultRuleTimeout, "Timeout maximum par règle (ex: 30s)")
	return cmd
}

type ruleRef struct {
	undo         string
	irreversible bool
}

// openOutputs ouvre la sortie composite (stdout + fichier journal). Si writeJournal
// est false (cas dry-run sans persistence), l'output va seulement sur stdout.
// Retourne le composite, le path du journal (vide si pas de fichier), et une erreur.
func openOutputs(runID string, writeJournal bool) (*compositeWriter, string, error) {
	if !writeJournal {
		return &compositeWriter{stdout: os.Stdout}, "", nil
	}
	journalDir := flagJournalDir
	if journalDir == "" {
		journalDir = journal.DefaultDir()
	}
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("mkdir %s: %w", journalDir, err)
	}
	path := filepath.Join(journalDir, runID+".ndjson")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open journal %s: %w", path, err)
	}
	return &compositeWriter{stdout: os.Stdout, file: f}, path, nil
}

// compositeWriter dual-écrit sur os.Stdout et un fichier journal optionnel.
//
// Politique défensive :
//   - Erreurs stdout ignorées : 'apply | head -10' ferme stdout au milieu du
//     run, ne doit PAS faire planter le moteur (le journal disque reste l'audit).
//   - Sync() après chaque write sur le fichier journal : un crash du process
//     préserve les events déjà émis (audit trail fiable même en cas de panic).
type compositeWriter struct {
	stdout *os.File
	file   *os.File
}

func (c *compositeWriter) Write(p []byte) (int, error) {
	_, _ = c.stdout.Write(p) // ignore broken pipe
	if c.file != nil {
		n, err := c.file.Write(p)
		if err != nil {
			return n, err
		}
		_ = c.file.Sync()
	}
	return len(p), nil
}

func (c *compositeWriter) Close() error {
	if c.file != nil {
		return c.file.Close()
	}
	return nil
}

func modeName(m executor.Mode) string {
	switch m {
	case executor.ModeDry:
		return "dry-run"
	case executor.ModeApply:
		return "apply"
	default:
		return "unknown"
	}
}

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func exitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exitError); ok {
		fmt.Fprintln(os.Stderr, ee.msg)
		return ee.code
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}
