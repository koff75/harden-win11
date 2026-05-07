// harden-engine est le moteur CLI v2 du projet harden-win11.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version est le numéro d'engine (override-able au build via -ldflags).
	Version = "0.1.0-dev"
	// ManifestVersion est la version du schéma manifest supportée.
	ManifestVersion = "1.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "harden-engine",
		Short: "Moteur de hardening Windows 11",
		Long:  "harden-engine — moteur de la baseline de sécurité Windows 11 v2.",
	}
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
