package ndjson

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriter_Emit_SingleEvent(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	event := map[string]any{
		"type":    "run_start",
		"run_id":  "2026-05-07T14-23-00",
		"dry_run": false,
	}

	if err := w.Emit(event); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	got := buf.String()
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
	// JSON content (any field order is OK, json.Marshal sorts map keys deterministically since Go 1.12)
	if !strings.Contains(got, `"type":"run_start"`) {
		t.Errorf("expected type=run_start in output, got %q", got)
	}
	if !strings.Contains(got, `"dry_run":false`) {
		t.Errorf("expected dry_run=false in output, got %q", got)
	}
}

func TestWriter_Emit_MultipleEvents_OnePerLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	_ = w.Emit(map[string]any{"type": "a"})
	_ = w.Emit(map[string]any{"type": "b"})
	_ = w.Emit(map[string]any{"type": "c"})

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d : %q", len(lines), buf.String())
	}
}
