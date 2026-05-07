package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultDir(t *testing.T) {
	dir := DefaultDir()
	if !strings.Contains(dir, "Harden-Win11") {
		t.Errorf("expected DefaultDir to contain Harden-Win11, got %q", dir)
	}
	if !strings.Contains(dir, "runs") {
		t.Errorf("expected DefaultDir to contain runs, got %q", dir)
	}
}

func TestListRuns_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	runs, err := ListRuns(dir)
	if err != nil {
		t.Fatalf("ListRuns on empty dir: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestListRuns_SortedByMtimeDesc(t *testing.T) {
	dir := t.TempDir()

	// Crée 3 fichiers avec des mtimes différents.
	files := []string{"old.ndjson", "middle.ndjson", "newest.ndjson"}
	for i, name := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mtime := time.Now().Add(time.Duration(i) * time.Second)
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	// Aussi un fichier non-ndjson qui doit être ignoré.
	if err := os.WriteFile(filepath.Join(dir, "ignore.txt"), []byte("noise"), 0o644); err != nil {
		t.Fatal(err)
	}

	runs, err := ListRuns(dir)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d : %v", len(runs), runs)
	}
	// L'ordre doit être : newest, middle, old (desc par mtime).
	if runs[0] != "newest" || runs[1] != "middle" || runs[2] != "old" {
		t.Errorf("expected [newest middle old], got %v", runs)
	}
}

func TestLatestRunID_Empty(t *testing.T) {
	dir := t.TempDir()
	_, err := LatestRunID(dir)
	if err == nil {
		t.Fatal("expected error on empty dir, got nil")
	}
}

func TestReadRun_ParsesNDJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"run_start","run_id":"abc"}
{"type":"action_result","rule_id":"foo.bar","status":"applied","before":{"x":1}}
{"type":"run_end","run_id":"abc"}
`
	path := filepath.Join(dir, "abc.ndjson")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := ReadRun(dir, "abc")
	if err != nil {
		t.Fatalf("ReadRun: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0]["type"] != "run_start" {
		t.Errorf("event 0 type: %v", events[0]["type"])
	}
	if events[1]["rule_id"] != "foo.bar" {
		t.Errorf("event 1 rule_id: %v", events[1]["rule_id"])
	}
	before, ok := events[1]["before"].(map[string]any)
	if !ok {
		t.Fatalf("event 1 before is not a map: %T", events[1]["before"])
	}
	if before["x"] != float64(1) {
		t.Errorf("event 1 before.x: %v (type %T)", before["x"], before["x"])
	}
}

func TestReadRun_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadRun(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error on missing run, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention run id, got %v", err)
	}
}

func TestReadRun_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"type":"run_start"}
{this is not valid JSON}
`
	if err := os.WriteFile(filepath.Join(dir, "corrupt.ndjson"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadRun(dir, "corrupt")
	if err == nil {
		t.Fatal("expected error on corrupted JSON, got nil")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected error to mention line number, got %v", err)
	}
}

func TestReadRun_LargeLine(t *testing.T) {
	// Vérifie que le scanner gère les lignes > 64KB (ex: ASR avec liste GUIDs).
	dir := t.TempDir()
	bigPayload := strings.Repeat("X", 100*1024) // 100KB
	content := `{"type":"run_start"}
{"type":"action_result","payload":"` + bigPayload + `"}
{"type":"run_end"}
`
	if err := os.WriteFile(filepath.Join(dir, "big.ndjson"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := ReadRun(dir, "big")
	if err != nil {
		t.Fatalf("ReadRun on large line: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}
