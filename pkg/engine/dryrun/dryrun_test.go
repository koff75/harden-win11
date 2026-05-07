package dryrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

func TestRun_DefenderRealtime(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}

	repo, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	manifestPath := filepath.Join(repo, "manifests", "01-defender.yaml")

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)

	opts := Options{
		ManifestDir: filepath.Join(repo, "manifests"),
		BasePath:    repo,
		Runner:      runner.New(),
		Writer:      w,
		RunID:       "test-run",
	}

	summary, err := Run(context.Background(), manifestPath, opts)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if summary.Skipped+summary.Applied+summary.Failed == 0 {
		t.Errorf("expected non-zero summary counters, got %+v", summary)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected >= 3 events (section_start + action_result + section_end), got %d : %s", len(lines), buf.String())
	}

	var sawSectionStart, sawSectionEnd, sawAction bool
	for _, line := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("invalid JSON line: %q : %v", line, err)
			continue
		}
		switch ev["type"] {
		case "section_start":
			sawSectionStart = true
			if ev["section_id"] != "defender" {
				t.Errorf("expected section_id=defender, got %v", ev["section_id"])
			}
		case "section_end":
			sawSectionEnd = true
		case "action_result":
			if ev["rule_id"] == "defender.realtime" {
				sawAction = true
				status, _ := ev["status"].(string)
				if status != "would_skip" && status != "would_apply" && status != "would_fail" {
					t.Errorf("unexpected status: %q", status)
				}
			}
		case "run_start", "run_end":
			t.Errorf("dryrun.Run() must NOT emit %s (caller's responsibility)", ev["type"])
		}
	}
	if !sawSectionStart {
		t.Error("did not see section_start event")
	}
	if !sawSectionEnd {
		t.Error("did not see section_end event")
	}
	if !sawAction {
		t.Error("did not see action_result for defender.realtime")
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
