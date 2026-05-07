// Package dryrun implémente la logique du dry-run : pour chaque règle,
// lancer .test.ps1 et émettre un event NDJSON would_apply/would_skip.
package dryrun

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/koff75/harden-win11/pkg/engine/manifest"
	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// Options configure une exécution dryrun.
type Options struct {
	ManifestDir string
	BasePath    string
	Runner      *runner.Runner
	Writer      *ndjson.Writer
	RunID       string
}

// Run exécute le dry-run sur toutes les règles du manifest fourni.
func Run(ctx context.Context, sectionPath string, opts Options) error {
	s, err := manifest.Load(sectionPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":             "run_start",
		"run_id":           opts.RunID,
		"manifest_version": s.Version,
		"dry_run":          true,
	})

	for _, rule := range s.Rules {
		testPath := filepath.Join(opts.BasePath, rule.Test)

		start := time.Now()
		out, err := opts.Runner.RunPS(ctx, testPath, nil)
		duration := time.Since(start)

		ev := map[string]any{
			"type":        "action_result",
			"run_id":      opts.RunID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"rule_id":     rule.ID,
			"duration_ms": duration.Milliseconds(),
			"dry_run":     true,
		}

		if err != nil {
			ev["status"] = "would_fail"
			ev["error"] = err.Error()
		} else if compliant, _ := out["compliant"].(bool); compliant {
			ev["status"] = "would_skip"
			ev["reason"] = "already_compliant"
			ev["current_state"] = out["current"]
		} else {
			ev["status"] = "would_apply"
			ev["current_state"] = out["current"]
		}

		_ = opts.Writer.Emit(ev)
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":   "run_end",
		"run_id": opts.RunID,
	})
	return nil
}
