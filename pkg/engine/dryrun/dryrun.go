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

// Run exécute le dry-run sur toutes les règles d'un manifest section.
//
// Émet un event "section_start" puis une suite "action_result" puis
// "section_end". Les events "run_start" et "run_end" englobants sont
// émis par le caller (CLI), pas ici, pour qu'un run multi-section ait
// 1 seul run_start/run_end et 1 paire section_start/section_end par section.
func Run(ctx context.Context, sectionPath string, opts Options) error {
	s, err := manifest.Load(sectionPath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	_ = opts.Writer.Emit(map[string]any{
		"type":             "section_start",
		"run_id":           opts.RunID,
		"section_id":       s.Section.ID,
		"section_order":    s.Section.Order,
		"section_title":    s.Section.Title,
		"manifest_version": s.Version,
		"rule_count":       len(s.Rules),
	})

	for _, rule := range s.Rules {
		testPath := filepath.Join(opts.BasePath, rule.Test)

		start := time.Now()
		out, err := opts.Runner.RunPS(ctx, testPath, nil)
		duration := time.Since(start)

		ev := map[string]any{
			"type":        "action_result",
			"run_id":      opts.RunID,
			"section_id":  s.Section.ID,
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
		"type":       "section_end",
		"run_id":     opts.RunID,
		"section_id": s.Section.ID,
	})
	return nil
}
