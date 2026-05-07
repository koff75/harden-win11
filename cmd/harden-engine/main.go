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
			var failed int
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				ext := filepath.Ext(e.Name())
				if ext != ".yaml" && ext != ".yml" {
					continue
				}
				path := filepath.Join(flagManifestDir, e.Name())
				if err := manifest.Validate(path, flagSchemaPath); err != nil {
					fmt.Fprintf(os.Stderr, "[FAIL] %s : %v\n", e.Name(), err)
					failed++
				} else {
					fmt.Fprintf(os.Stderr, "[OK]   %s\n", e.Name())
				}
			}
			if failed > 0 {
				return &exitError{code: 3, msg: fmt.Sprintf("%d manifests invalid", failed)}
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
			for _, e := range entries {
				if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
					continue
				}
				p := filepath.Join(flagManifestDir, e.Name())
				s, err := manifest.Load(p)
				if err != nil {
					continue
				}
				if flagSection != "" && s.Section.ID != flagSection {
					continue
				}
				sections = append(sections, sec{path: p, order: s.Section.Order, id: s.Section.ID})
			}

			if len(sections) == 0 {
				if flagSection != "" {
					return fmt.Errorf("section %q not found in %s", flagSection, flagManifestDir)
				}
				return fmt.Errorf("no manifests found in %s", flagManifestDir)
			}

			sort.Slice(sections, func(i, j int) bool { return sections[i].order < sections[j].order })

			absManifestDir, _ := filepath.Abs(flagManifestDir)
			base := filepath.Dir(absManifestDir)

			runID := time.Now().UTC().Format("2006-01-02T15-04-05")
			w := ndjson.NewWriter(os.Stdout)
			ctx := context.Background()

			for _, sct := range sections {
				if err := dryrun.Run(ctx, sct.path, dryrun.Options{
					ManifestDir: flagManifestDir,
					BasePath:    base,
					Runner:      runner.New(),
					Writer:      w,
					RunID:       runID,
				}); err != nil {
					return fmt.Errorf("section %s: %w", sct.id, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Mode dry-run (rien d'exécuté)")
	cmd.Flags().StringVar(&flagSection, "section", "", "ID de la section à dry-runner (vide = toutes)")
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
