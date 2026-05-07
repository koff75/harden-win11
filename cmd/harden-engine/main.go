// harden-engine est le moteur CLI v2 du projet harden-win11.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/dryrun"
	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
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
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Moteur de hardening Windows 11",
		Long:  "harden-engine — moteur de la baseline de sécurité Windows 11 v2.",
	}
	rootCmd.PersistentFlags().StringVar(&flagManifestDir, "manifest-dir", "manifests", "Dossier contenant les manifests YAML")
	rootCmd.PersistentFlags().StringVar(&flagSchemaPath, "schema", "schemas/manifest.schema.json", "Chemin du JSONSchema")

	rootCmd.AddCommand(versionCmd(), validateCmd(), applyCmd())

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
			entries, err := os.ReadDir(flagManifestDir)
			if err != nil {
				return fmt.Errorf("read manifest dir: %w", err)
			}

			validator, err := manifest.NewValidator(flagSchemaPath)
			if err != nil {
				return &exitError{code: 4, msg: err.Error()}
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
				path := filepath.Join(flagManifestDir, e.Name())
				if err := validator.ValidateFile(path); err != nil {
					fmt.Fprintf(os.Stderr, "[FAIL] %s : %v\n", e.Name(), err)
					failed++
				} else {
					fmt.Fprintf(os.Stderr, "[OK]   %s\n", e.Name())
					validPaths = append(validPaths, path)
				}
			}

			if processed == 0 {
				return &exitError{code: 4, msg: fmt.Sprintf("no manifests found in %s (expected *.yaml or *.yml)", flagManifestDir)}
			}

			// Détecte les collisions de section.id entre les manifests valides.
			seen := map[string]string{}
			var collisions int
			for _, p := range validPaths {
				s, err := manifest.Load(p)
				if err != nil {
					continue
				}
				if existing, ok := seen[s.Section.ID]; ok {
					fmt.Fprintf(os.Stderr, "[COLLISION] section.id %q is defined in both %s and %s\n",
						s.Section.ID, filepath.Base(existing), filepath.Base(p))
					collisions++
				} else {
					seen[s.Section.ID] = p
				}
			}

			if failed > 0 {
				return &exitError{code: 3, msg: fmt.Sprintf("%d manifests invalid", failed)}
			}
			if collisions > 0 {
				return &exitError{code: 3, msg: fmt.Sprintf("%d section.id collisions", collisions)}
			}
			return nil
		},
	}
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Exécute (ou dry-run) les règles. Sans --section = toutes les sections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flagDryRun {
				return fmt.Errorf("only --dry-run is supported in this walking skeleton (use --dry-run)")
			}
			if flagRuleTimeout < 0 {
				return &exitError{code: 4, msg: "--rule-timeout must be >= 0 (use 0 to keep the default 30s)"}
			}

			entries, err := os.ReadDir(flagManifestDir)
			if err != nil {
				return fmt.Errorf("read manifest dir: %w", err)
			}

			type sec struct {
				path  string
				order int
				id    string
			}
			var sections []sec
			seenIDs := map[string]string{}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(e.Name()))
				if ext != ".yaml" && ext != ".yml" {
					continue
				}
				p := filepath.Join(flagManifestDir, e.Name())
				s, err := manifest.Load(p)
				if err != nil {
					continue
				}
				if existing, ok := seenIDs[s.Section.ID]; ok {
					return &exitError{code: 3, msg: fmt.Sprintf("section.id %q is defined in both %s and %s — please fix before running apply",
						s.Section.ID, filepath.Base(existing), filepath.Base(p))}
				}
				seenIDs[s.Section.ID] = p
				if flagSection != "" && s.Section.ID != flagSection {
					continue
				}
				sections = append(sections, sec{path: p, order: s.Section.Order, id: s.Section.ID})
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
				// Tie-breaker déterministe : alphabétique sur l'id de section.
				return sections[i].id < sections[j].id
			})

			absManifestDir, _ := filepath.Abs(flagManifestDir)
			base := filepath.Dir(absManifestDir)

			runID := time.Now().UTC().Format("2006-01-02T15-04-05")
			w := ndjson.NewWriter(os.Stdout)
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
				"dry_run":          true,
				"section_count":    len(sections),
				"sections":         sectionIDs,
			})

			var total dryrun.Summary
			for _, sct := range sections {
				summary, err := dryrun.Run(ctx, sct.path, dryrun.Options{
					ManifestDir: flagManifestDir,
					BasePath:    base,
					Runner:      runner.New(),
					Writer:      w,
					RunID:       runID,
					RuleTimeout: flagRuleTimeout,
				})
				if err != nil {
					_ = w.Emit(map[string]any{
						"type":   "run_end",
						"run_id": runID,
						"error":  err.Error(),
					})
					return fmt.Errorf("section %s: %w", sct.id, err)
				}
				total.Skipped += summary.Skipped
				total.Applied += summary.Applied
				total.Failed += summary.Failed
			}

			_ = w.Emit(map[string]any{
				"type":    "run_end",
				"run_id":  runID,
				"skipped": total.Skipped,
				"applied": total.Applied,
				"failed":  total.Failed,
			})

			// Exit code 2 si au moins une règle a planté (would_fail) — utile pour scripting.
			if total.Failed > 0 {
				return &exitError{code: 2, msg: fmt.Sprintf("%d rules failed during dry-run (would_fail)", total.Failed)}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Mode dry-run (rien d'exécuté)")
	cmd.Flags().StringVar(&flagSection, "section", "", "ID de la section à dry-runner (vide = toutes)")
	cmd.Flags().DurationVar(&flagRuleTimeout, "rule-timeout", dryrun.DefaultRuleTimeout, "Timeout maximum par règle (ex: 30s, 1m)")
	return cmd
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
