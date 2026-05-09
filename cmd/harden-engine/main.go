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

	"github.com/koff75/harden-win11/pkg/engine/baseline"
	"github.com/koff75/harden-win11/pkg/engine/executor"
	"github.com/koff75/harden-win11/pkg/engine/journal"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/restorepoint"
	"github.com/koff75/harden-win11/pkg/engine/runner"
	"github.com/koff75/harden-win11/pkg/engine/snapshot"
	"github.com/koff75/harden-win11/pkg/engine/watchlist"
	"github.com/koff75/harden-win11/pkg/engine/winadmin"
	"github.com/spf13/cobra"
)

var (
	Version         = "0.1.0-dev"
	ManifestVersion = "1.0"
)

var (
	flagManifestDir      string
	flagSchemaPath       string
	flagDryRun           bool
	flagSection          string
	flagRuleTimeout      time.Duration
	flagJournalDir       string
	flagYes              bool
	flagRunID            string
	flagRuleID           string
	flagProfile          string
	flagAudit            bool
	flagParallel         int
	flagSkipRestorePoint bool
	flagSeverity         string
	flagSkipSnapshot     bool
	flagSkipWatchlist    bool
	flagUndoSince        time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Windows 11 hardening engine",
		Long:  "harden-engine — moteur de la baseline de sécurité Windows 11 v2.",
	}
	rootCmd.PersistentFlags().StringVar(&flagManifestDir, "manifest-dir", "manifests", "Folder containing the YAML manifests")
	rootCmd.PersistentFlags().StringVar(&flagSchemaPath, "schema", "schemas/manifest.schema.json", "Path to the JSONSchema")
	rootCmd.PersistentFlags().StringVar(&flagJournalDir, "journal-dir", "", "NDJSON journal folder (empty = %ProgramData%\\Harden-Win11\\runs\\)")

	rootCmd.AddCommand(versionCmd(), validateCmd(), applyCmd(), undoCmd(), coverageCmd(), snapshotCmd(), watchEventsCmd(), watchlistCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print engine + manifest + OS version (JSON)",
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
		Short: "Validate all manifests against the JSONSchema",
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

	// Collision detection sur section.id ET sur rule.id global (cross-fichiers).
	seenSection := map[string]string{}
	seenRule := map[string]string{} // rule.id → file qui le définit
	var collisions int
	var sections []manifestEntry
	for _, p := range validPaths {
		s, err := manifest.Load(p)
		if err != nil {
			continue
		}
		if existing, ok := seenSection[s.Section.ID]; ok {
			if verbose {
				fmt.Fprintf(os.Stderr, "[COLLISION] section.id %q is defined in both %s and %s\n",
					s.Section.ID, filepath.Base(existing), filepath.Base(p))
			}
			collisions++
			continue // skip cette section pour ne pas pollluer le check rule.id
		}
		seenSection[s.Section.ID] = p

		for _, r := range s.Rules {
			if existingFile, ok := seenRule[r.ID]; ok {
				if verbose {
					fmt.Fprintf(os.Stderr, "[COLLISION] rule.id %q is defined in both %s and %s\n",
						r.ID, filepath.Base(existingFile), filepath.Base(p))
				}
				collisions++
			} else {
				seenRule[r.ID] = p
			}
		}
		sections = append(sections, manifestEntry{path: p, order: s.Section.Order, id: s.Section.ID})
	}

	if failed > 0 {
		return nil, failed, &exitError{code: 3, msg: fmt.Sprintf("%d manifests invalid", failed)}
	}
	if collisions > 0 {
		return nil, collisions, &exitError{code: 3, msg: fmt.Sprintf("%d collisions detected", collisions)}
	}
	return sections, 0, nil
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run rules in real or dry-run mode. Without --section = all sections",
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

			// Snapshot pre-apply : capture l'état système pertinent pour debug
			// post-incident. Best-effort, ne plante jamais l'apply.
			if mode == executor.ModeApply && !flagSkipSnapshot {
				if path, err := snapshot.Capture(ctx, runID, snapshot.PhasePre, 30*time.Second); err == nil {
					_ = w.Emit(map[string]any{
						"type":   "snapshot",
						"run_id": runID,
						"phase":  "pre",
						"path":   path,
					})
				} else {
					fmt.Fprintf(os.Stderr, "Snapshot pre-apply : %v (continue)\n", err)
				}
			}

			// Restore Point Windows : ceinture-bretelles avant un apply réel.
			// Best-effort : si ça plante, on log et on continue (le journal NDJSON +
			// .undo.ps1 restent les voies principales de rollback).
			if mode == executor.ModeApply && !flagSkipRestorePoint {
				fmt.Fprintln(os.Stderr, "Création d'un Windows System Restore Point (peut prendre 30-60s)...")
				st := restorepoint.Create(ctx, runID, 90*time.Second)
				rpEv := map[string]any{
					"type":        "restore_point",
					"run_id":      runID,
					"created":     st.Created,
					"description": st.Description,
					"duration_ms": st.Duration.Milliseconds(),
				}
				if !st.Created {
					rpEv["reason"] = st.Reason
					rpEv["error"] = st.Error
					fmt.Fprintf(os.Stderr, "Note : Restore Point non créé (%s) — l'apply continue (rollback via journal NDJSON disponible).\n", st.Reason)
				} else {
					fmt.Fprintln(os.Stderr, "Restore Point créé.")
				}
				_ = w.Emit(rpEv)
			}

			var total executor.Summary
			var aborted bool
			var abortedSection string
			for _, sct := range sections {
				r := runner.New()
				if flagAudit {
					r.Env = map[string]string{"HARDEN_ASR_MODE": "audit"}
				}
				summary, err := executor.Run(ctx, sct.path, executor.Options{
					Mode:        mode,
					ManifestDir: flagManifestDir,
					BasePath:    base,
					Runner:      r,
					Writer:      w,
					RunID:       runID,
					RuleTimeout: flagRuleTimeout,
					Profile:     flagProfile,
					Parallel:    flagParallel,
					Severity:    flagSeverity,
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

			// Snapshot post-apply : capture l'état après les modifications.
			// Diffable contre le pre-apply via 'harden-engine snapshot diff <runID>'.
			if mode == executor.ModeApply && !flagSkipSnapshot {
				if path, err := snapshot.Capture(ctx, runID, snapshot.PhasePost, 30*time.Second); err == nil {
					_ = w.Emit(map[string]any{
						"type":   "snapshot",
						"run_id": runID,
						"phase":  "post",
						"path":   path,
					})
				}
			}

			// Watchlist 24h : enregistre une tâche planifiée Windows qui va
			// surveiller Event Viewer pendant 24h. Best-effort. L'utilisateur
			// peut désactiver via --no-watchlist ou supprimer la tâche
			// manuellement (Task Scheduler → harden-watchlist-<runID>).
			if mode == executor.ModeApply && !flagSkipWatchlist {
				// Auto-baseline : si pas de baseline ou > 30 jours, apprendre
				// maintenant pour que la watchlist 24h ait des seuils adaptatifs.
				bl, _ := watchlist.LoadBaseline()
				needsLearn := bl == nil
				if bl != nil {
					if t, err := time.Parse(time.RFC3339, bl.LearnedAt); err == nil {
						if time.Since(t) > 30*24*time.Hour {
							needsLearn = true
						}
					}
				}
				if needsLearn {
					fmt.Fprintln(os.Stderr, "Apprentissage de la baseline Event Viewer (peut prendre 1-2 min, exécuté une fois par mois)…")
					if newBl, err := watchlist.Learn(ctx, watchlist.DefaultSources, 7); err == nil {
						_ = watchlist.SaveBaseline(newBl)
						fmt.Fprintf(os.Stderr, "Baseline : %d source(s) apprises sur 7 jours.\n", len(newBl.Sources))
					}
				}
				if exe, err := os.Executable(); err == nil {
					if err := watchlist.ScheduleTask(ctx, runID, exe, 5, 24); err != nil {
						fmt.Fprintf(os.Stderr, "Watchlist 24h non programmée : %v (continue)\n", err)
					} else {
						fmt.Fprintln(os.Stderr, "Watchlist 24h programmée (tâche planifiée harden-watchlist-"+runID+").")
						_ = w.Emit(map[string]any{
							"type":   "watchlist_scheduled",
							"run_id": runID,
							"task":   "harden-watchlist-" + runID,
						})
					}
				}
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
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Dry-run mode (reads state, modifies nothing)")
	cmd.Flags().StringVar(&flagSection, "section", "", "Section ID to run (empty = all sections)")
	cmd.Flags().StringVar(&flagProfile, "profile", "", "Risk profile (personal | business | maximal). Empty = all rules.")
	cmd.Flags().BoolVar(&flagAudit, "audit", false, "Audit mode for ASR / Network Protection (does not apply, only logs events)")
	cmd.Flags().IntVar(&flagParallel, "parallel", 1, "Number of dry-run rules executed in parallel (default 1 = sequential). No effect on real apply.")
	cmd.Flags().BoolVar(&flagSkipRestorePoint, "skip-restore-point", false, "Skip the Windows System Restore Point creation before apply (default: created — last-resort safety net).")
	cmd.Flags().StringVar(&flagSeverity, "severity", "", "Filter by severity (critical | important | nice-to-have). Lets you apply in waves: --severity critical first, reboot, then --severity important.")
	cmd.Flags().BoolVar(&flagSkipSnapshot, "no-snapshot", false, "Skip pre/post-apply snapshot capture (default: captured to %ProgramData%\\Harden-Win11\\snapshots\\, useful for post-incident debug).")
	cmd.Flags().BoolVar(&flagSkipWatchlist, "no-watchlist", false, "Skip the 24h Event Viewer watchlist scheduling (default: registers a scheduled task that detects post-apply anomalies).")
	cmd.Flags().DurationVar(&flagRuleTimeout, "rule-timeout", executor.DefaultRuleTimeout, "Per-rule timeout (e.g., 30s, 1m)")
	cmd.Flags().BoolVar(&flagYes, "yes", false, "Skip the interactive confirmation before real apply")
	return cmd
}

func watchlistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watchlist",
		Short: "Manage the adaptive watchlist baseline (median + σ per Event Viewer source)",
	}

	var daysBack int
	learnCmd := &cobra.Command{
		Use:   "baseline-learn",
		Short: "Learn baseline from Event Viewer history (default last 7 days)",
		Long: `Reads Get-WinEvent day by day on watched sources (SMB, Defender, NetBT, Schannel, PrintService) and computes median + stddev. Persisted in %ProgramData%\Harden-Win11\watchlist\baseline.json.

Without baseline: static thresholds (5-20 events per source). With: dynamic thresholds ` + "`max(static, median + 3σ)`" + `. A noisy machine (legacy NAS generating 50 SMB errors/day) no longer alerts on every event; a quiet machine keeps the minimum threshold.`,
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Apprentissage baseline sur %d jours d'historique Event Viewer (peut prendre 1-2 min)…\n", daysBack)
			bl, err := watchlist.Learn(context.Background(), watchlist.DefaultSources, daysBack)
			if err != nil {
				return err
			}
			if err := watchlist.SaveBaseline(bl); err != nil {
				return err
			}
			fmt.Printf("Baseline apprise : %d source(s), %d échantillons, fenêtre %d jours.\n", len(bl.Sources), bl.SamplesUsed, bl.WindowDays)
			for k, s := range bl.Sources {
				fmt.Printf("  %s : médiane=%.1f σ=%.1f max=%d (%d jours)\n", k, s.Median, s.Stddev, s.Max, len(s.DailyCounts))
			}
			return nil
		},
	}
	learnCmd.Flags().IntVar(&daysBack, "days", 7, "Nombre de jours d'historique à analyser (default 7)")

	showCmd := &cobra.Command{
		Use:   "baseline-show",
		Short: "Print the current baseline (JSON)",
		RunE: func(c *cobra.Command, args []string) error {
			bl, err := watchlist.LoadBaseline()
			if err != nil {
				return err
			}
			if bl == nil {
				fmt.Println("Aucune baseline. Lance 'watchlist baseline-learn' pour en créer une.")
				return nil
			}
			out, _ := json.MarshalIndent(bl, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}

	clearCmd := &cobra.Command{
		Use:   "baseline-clear",
		Short: "Delete the baseline (forces fallback to static thresholds)",
		RunE: func(c *cobra.Command, args []string) error {
			path := watchlist.BaselinePath()
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Println("Baseline supprimée.")
			return nil
		},
	}

	cmd.AddCommand(learnCmd, showCmd, clearCmd)
	return cmd
}

func watchEventsCmd() *cobra.Command {
	var (
		runID     string
		duration  time.Duration
		pollEvery time.Duration
		sinceISO  string
	)
	cmd := &cobra.Command{
		Use:   "watch-events",
		Short: "Watch Event Viewer for N hours to detect post-apply functional breakage",
		Long: `Runs a Get-WinEvent polling loop on the sensitive sources
(SMB, Defender, NetBT, Schannel, PrintService) to detect anomalies that
appear after an apply. Writes alerts to
%ProgramData%\Harden-Win11\watchlist\<runID>.json. The GUI reads this
folder at boot and shows a banner if it sees recent alerts.

Typically registered as a scheduled task by real 'apply', but can also be
run manually: harden-engine watch-events --run-id myrun --duration 24h`,
		RunE: func(c *cobra.Command, args []string) error {
			if runID == "" {
				return &exitError{code: 4, msg: "--run-id requis"}
			}
			since := time.Now().Add(-1 * time.Minute)
			if sinceISO != "" {
				t, err := time.Parse(time.RFC3339, sinceISO)
				if err != nil {
					return &exitError{code: 4, msg: fmt.Sprintf("--since must be RFC3339: %v", err)}
				}
				since = t
			}
			fmt.Fprintf(os.Stderr, "watch-events run=%s baseline=%s duration=%s poll=%s\n",
				runID, since.Format(time.RFC3339), duration, pollEvery)
			ctx := context.Background()
			report, err := watchlist.Watch(ctx, runID, since, duration, pollEvery)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "watch done: %d alerts after %d polls\n", len(report.Alerts), report.Polls)
			b, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "Identifiant du run associé (sert de clé pour le fichier report)")
	cmd.Flags().DurationVar(&duration, "duration", 24*time.Hour, "Durée totale de surveillance (default 24h)")
	cmd.Flags().DurationVar(&pollEvery, "poll", 1*time.Hour, "Intervalle entre 2 scans Get-WinEvent")
	cmd.Flags().StringVar(&sinceISO, "since", "", "Timestamp RFC3339 à partir duquel lire les events (default = 1min avant le démarrage)")
	return cmd
}

func snapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture / diff system snapshots (post-apply debug)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "capture <runID>",
		Short: "Capture an ad-hoc snapshot (without apply)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			path, err := snapshot.Capture(context.Background(), args[0], snapshot.PhasePre, 60*time.Second)
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "diff <runID>",
		Short: "Show the changes between pre and post snapshots of a run",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			runID := args[0]
			pre, err := snapshot.LoadSnapshot(snapshot.Path(runID, snapshot.PhasePre))
			if err != nil {
				return fmt.Errorf("load pre snapshot: %w", err)
			}
			post, err := snapshot.LoadSnapshot(snapshot.Path(runID, snapshot.PhasePost))
			if err != nil {
				return fmt.Errorf("load post snapshot: %w", err)
			}
			diffs := snapshot.Diff(pre, post)
			if len(diffs) == 0 {
				fmt.Println("Aucune différence entre pre et post.")
				return nil
			}
			fmt.Printf("%d changement(s) entre %s/pre et /post :\n", len(diffs), runID)
			for _, d := range diffs {
				fmt.Printf("  [%s/%s] %s : %v → %v\n", d.Kind, d.Change, d.Key, d.Before, d.After)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available snapshots",
		RunE: func(c *cobra.Command, args []string) error {
			dir := snapshot.DefaultDir()
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("Aucun snapshot dans %s\n", dir)
					return nil
				}
				return err
			}
			fmt.Printf("Snapshots dans %s :\n", dir)
			for _, e := range entries {
				if !e.IsDir() {
					fmt.Printf("  %s\n", e.Name())
				}
			}
			return nil
		},
	})
	return cmd
}

func coverageCmd() *cobra.Command {
	var (
		mappingPath string
		jsonOut     bool
	)
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Show rule coverage vs CIS / ANSSI / MS Security Baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := baseline.Load(mappingPath)
			if err != nil {
				return &exitError{code: 4, msg: err.Error()}
			}
			entries, _, err := loadAndValidateManifests(flagManifestDir, flagSchemaPath, false)
			if err != nil {
				return err
			}
			ruleIDs, err := collectRuleIDs(entries)
			if err != nil {
				return err
			}
			rep := baseline.Compute(doc, ruleIDs)
			if jsonOut {
				b, _ := json.MarshalIndent(rep, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			printCoverageText(rep)
			return nil
		},
	}
	cmd.Flags().StringVar(&mappingPath, "mapping", "mappings/baselines.yaml", "Chemin du fichier de mapping baseline")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Sortie JSON (vs texte humain)")
	return cmd
}

func collectRuleIDs(entries []manifestEntry) ([]string, error) {
	ids := make([]string, 0, 128)
	for _, e := range entries {
		sec, err := manifest.Load(e.path)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", e.path, err)
		}
		for _, r := range sec.Rules {
			ids = append(ids, r.ID)
		}
	}
	return ids, nil
}

func printCoverageText(rep *baseline.CoverageReport) {
	fmt.Printf("Total règles harden-win11 : %d\n", rep.TotalRules)
	fmt.Printf("Règles avec ≥1 mapping    : %d (%d%%)\n",
		rep.MappedRules, pct(rep.MappedRules, rep.TotalRules))
	fmt.Println()
	order := []string{"cis", "anssi", "ms_baseline"}
	for _, k := range order {
		st := rep.Frameworks[k]
		fmt.Printf("[%s]\n", st.Framework)
		fmt.Printf("  Règles couvertes : %d / %d (%d%%)\n",
			st.Mapped, rep.TotalRules, pct(st.Mapped, rep.TotalRules))
		fmt.Printf("  Contrôles uniques cités : %d\n", st.UniqueControls)
		if len(st.SampleControls) > 0 {
			fmt.Printf("  Exemples : %v\n", st.SampleControls)
		}
		fmt.Printf("  Règles sans mapping ce framework : %d\n", len(st.UnmappedRules))
		fmt.Println()
	}
	if rep.Disclaimer != "" {
		fmt.Println("⚠ ", strings.TrimSpace(rep.Disclaimer))
	}
}

func pct(num, denom int) int {
	if denom == 0 {
		return 0
	}
	return int(float64(num) / float64(denom) * 100)
}

func undoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo",
		Short: "Restore state before a run via .undo.ps1 (reads NDJSON journal)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagRuleTimeout < 0 {
				return &exitError{code: 4, msg: "--rule-timeout must be >= 0"}
			}

			journalDir := flagJournalDir
			if journalDir == "" {
				journalDir = journal.DefaultDir()
			}

			// Résoudre quels runs charger : --since prime sur --run-id.
			var runIDsToProcess []string
			if flagUndoSince > 0 {
				cutoff := time.Now().Add(-flagUndoSince)
				ids, err := journal.RunsModifiedSince(journalDir, cutoff)
				if err != nil {
					return &exitError{code: 4, msg: fmt.Sprintf("list runs since %s: %v", flagUndoSince, err)}
				}
				if len(ids) == 0 {
					return &exitError{code: 4, msg: fmt.Sprintf("no runs found since %s ago in %s", flagUndoSince, journalDir)}
				}
				runIDsToProcess = ids
				fmt.Fprintf(os.Stderr, "Time-aware rollback : %d run(s) trouvés dans la fenêtre des %s.\n", len(ids), flagUndoSince)
			} else {
				runIDToUse := flagRunID
				if runIDToUse == "" {
					latest, err := journal.LatestRunID(journalDir)
					if err != nil {
						return &exitError{code: 4, msg: fmt.Sprintf("find latest run: %v", err)}
					}
					runIDToUse = latest
				}
				runIDsToProcess = []string{runIDToUse}
			}

			// Aggregate les rules applied à travers tous les runs (LIFO : run le
			// plus récent d'abord, dans chaque run la dernière rule en premier).
			// Si une rule a été applied puis re-applied après un undo, on prend
			// la version la plus récente seulement (déduplication par ruleID).
			seenRules := map[string]bool{}
			var toUndo []journal.AppliedRule
			runIDByRule := map[string]string{}

			for _, rid := range runIDsToProcess {
				events, err := journal.ReadRun(journalDir, rid)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: skip run %s (read error: %v)\n", rid, err)
					continue
				}
				// Iter en reverse pour respect LIFO local du run.
				for i := len(events) - 1; i >= 0; i-- {
					ev := events[i]
					if ev["type"] != "action_result" || ev["status"] != "applied" {
						continue
					}
					ruleID, _ := ev["rule_id"].(string)
					if seenRules[ruleID] {
						continue
					}
					if flagRuleID != "" && ruleID != flagRuleID {
						continue
					}
					sectionID, _ := ev["section_id"].(string)
					seenRules[ruleID] = true
					runIDByRule[ruleID] = rid
					toUndo = append(toUndo, journal.AppliedRule{
						RuleID:    ruleID,
						SectionID: sectionID,
						Before:    ev["before"],
					})
				}
			}

			if len(toUndo) == 0 {
				if flagRuleID != "" {
					return &exitError{code: 4, msg: fmt.Sprintf("no applied rule %q found in selected runs", flagRuleID)}
				}
				return &exitError{code: 4, msg: "no applied rules found in selected run(s)"}
			}

			// Pour la confirmation, on utilise un id synthétique ou le seul run.
			runIDToUse := runIDsToProcess[0]
			if len(runIDsToProcess) > 1 {
				runIDToUse = fmt.Sprintf("[%d runs since %s]", len(runIDsToProcess), flagUndoSince)
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
				undoPath := ref.undo
				if !filepath.IsAbs(undoPath) {
					undoPath = filepath.Join(base, undoPath)
				}
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
				"type":   "run_end",
				"run_id": "undo-" + undoRunID,
				"ok":     ok,
				"failed": failed,
			})

			if failed > 0 {
				return &exitError{code: 2, msg: fmt.Sprintf("%d undo operations failed", failed)}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&flagRunID, "run-id", "", "Run ID dont undo les règles (vide = dernier run)")
	cmd.Flags().DurationVar(&flagUndoSince, "since", 0, "Undo toutes les rules applied dans les N dernières heures/jours (ex: 24h, 7d=168h). Override --run-id : agrège LIFO sur tous les runs de la fenêtre, déduplique par rule_id.")
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
