package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// TestRun_ApplyMode_Skipped vérifie qu'en ModeApply, une rule already-compliant
// est skipped (pas d'action.ps1 lancée).
func TestRun_ApplyMode_Skipped(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-skipped.yaml")

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)
	summary, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   w,
		RunID:    "test-apply-skipped",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Skipped != 1 || summary.Applied != 0 || summary.Failed != 0 {
		t.Errorf("expected Skipped=1 Applied=0 Failed=0, got %+v", summary)
	}

	events := parseEvents(t, buf.Bytes())
	expectStatus(t, events, "fixture.always_ok", "skipped")
}

// TestRun_ApplyMode_Applied vérifie qu'une rule non-conforme avec action OK
// passe en status=applied.
func TestRun_ApplyMode_Applied(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-applied.yaml")

	// Marker file pour la fixture stateful — cleanup avant + après pour idempotence.
	markerPath := filepath.Join(t.TempDir(), "stateful.marker")
	t.Setenv("HARDEN_TEST_MARKER", markerPath)
	_ = os.Remove(markerPath)

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)
	r := runner.New()
	r.Env = map[string]string{"HARDEN_TEST_MARKER": markerPath}
	summary, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   r,
		Writer:   w,
		RunID:    "test-apply-applied",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.Applied != 1 || summary.Skipped != 0 || summary.Failed != 0 {
		t.Errorf("expected Applied=1 Skipped=0 Failed=0, got %+v", summary)
	}

	events := parseEvents(t, buf.Bytes())
	expectStatus(t, events, "fixture.success", "applied")

	// L'event doit contenir before et after de l'action.
	for _, ev := range events {
		if ev["rule_id"] == "fixture.success" && ev["type"] == "action_result" {
			if ev["before"] == nil {
				t.Error("expected 'before' field in applied event")
			}
			if ev["after"] == nil {
				t.Error("expected 'after' field in applied event")
			}
		}
	}
}

// TestRun_ApplyMode_RecheckFails : l'action retourne ok=true mais le re-test
// post-apply dit non-conforme (action menteuse / GPO override). Le moteur
// doit déclencher un rollback comme si l'action avait planté.
func TestRun_ApplyMode_RecheckFails(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-recheck-fails.yaml")

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)
	summary, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   w,
		RunID:    "test-apply-recheck-fails",
	})
	if !errors.Is(err, ErrAborted) {
		t.Fatalf("expected ErrAborted, got %v", err)
	}
	if summary.RolledBack != 1 {
		t.Errorf("expected RolledBack=1 (recheck triggered rollback), got %+v", summary)
	}
	events := parseEvents(t, buf.Bytes())
	var sawNonCompliant bool
	for _, ev := range events {
		if ev["type"] == "action_result" && ev["rule_id"] == "fixture.lying_action" {
			if ev["recheck"] == "non_compliant" {
				sawNonCompliant = true
			}
		}
	}
	if !sawNonCompliant {
		t.Error("expected recheck=non_compliant on action_result event")
	}
}

// TestRun_ApplyMode_RollbackOnFail vérifie qu'une action qui plante déclenche
// le rollback via .undo.ps1, émet un rollback_result, et retourne ErrAborted.
func TestRun_ApplyMode_RollbackOnFail(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-rollback.yaml")

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)
	summary, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   w,
		RunID:    "test-apply-rollback",
	})
	if !errors.Is(err, ErrAborted) {
		t.Fatalf("expected ErrAborted, got %v", err)
	}
	if summary.RolledBack != 1 {
		t.Errorf("expected RolledBack=1, got %+v", summary)
	}
	if summary.Failed != 0 {
		t.Errorf("expected Failed=0 (rollback succeeded), got %+v", summary)
	}

	events := parseEvents(t, buf.Bytes())

	var sawActionFail, sawRollback, sawSectionEndAborted bool
	for _, ev := range events {
		if ev["type"] == "action_result" && ev["rule_id"] == "fixture.fail" {
			sawActionFail = true
			if ev["status"] != "rolled_back" {
				t.Errorf("expected status=rolled_back, got %v", ev["status"])
			}
		}
		if ev["type"] == "rollback_result" && ev["rule_id"] == "fixture.fail" {
			sawRollback = true
			if ev["status"] != "rollback_ok" {
				t.Errorf("expected rollback status=rollback_ok, got %v", ev["status"])
			}
		}
		if ev["type"] == "section_end" && ev["aborted"] == true {
			sawSectionEndAborted = true
		}
	}
	if !sawActionFail {
		t.Error("did not see action_result for fixture.fail")
	}
	if !sawRollback {
		t.Error("did not see rollback_result event")
	}
	if !sawSectionEndAborted {
		t.Error("did not see section_end with aborted=true")
	}
}

// helpers

func parseEvents(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var events []map[string]any
	for i, line := range lines {
		if line == "" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d: %v (%q)", i, err, line)
			continue
		}
		events = append(events, ev)
	}
	return events
}

func expectStatus(t *testing.T, events []map[string]any, ruleID, expected string) {
	t.Helper()
	for _, ev := range events {
		if ev["type"] == "action_result" && ev["rule_id"] == ruleID {
			if ev["status"] != expected {
				t.Errorf("rule %s : expected status=%s, got %v", ruleID, expected, ev["status"])
			}
			return
		}
	}
	t.Errorf("rule %s : no action_result event found", ruleID)
}

func mustFindRepo(t *testing.T) string {
	t.Helper()
	r, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo: %v", err)
	}
	return r
}
