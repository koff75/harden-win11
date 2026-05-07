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

	if err := Run(context.Background(), manifestPath, opts); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected >= 3 events, got %d : %s", len(lines), buf.String())
	}

	var sawAction bool
	for _, line := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("invalid JSON line: %q : %v", line, err)
			continue
		}
		if ev["type"] == "action_result" && ev["rule_id"] == "defender.realtime" {
			sawAction = true
			status, _ := ev["status"].(string)
			if status != "would_skip" && status != "would_apply" && status != "would_fail" {
				t.Errorf("unexpected status: %q", status)
			}
		}
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
