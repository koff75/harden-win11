package ndjson

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

// TestWriter_ConcurrentEmit_ThreadSafe vérifie que Emit est thread-safe :
// avec N goroutines × M events, on doit obtenir exactement N*M lignes,
// chacune un JSON valide (pas d'interleaving qui casserait le JSON).
func TestWriter_ConcurrentEmit_ThreadSafe(t *testing.T) {
	const numGoroutines = 50
	const eventsPerGoroutine = 100

	var buf bytes.Buffer
	w := NewWriter(&buf)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			payload := strings.Repeat("X", 200)
			for j := 0; j < eventsPerGoroutine; j++ {
				_ = w.Emit(map[string]any{
					"goroutine": id,
					"seq":       j,
					"payload":   payload,
				})
			}
		}(i)
	}
	wg.Wait()

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	expected := numGoroutines * eventsPerGoroutine

	if len(lines) != expected {
		t.Fatalf("expected %d lines, got %d (interleaving detected — Writer not thread-safe)",
			expected, len(lines))
	}

	// Chaque ligne doit être un JSON valide complet.
	for i, line := range lines {
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d is not valid JSON (interleaved write?): %q : %v", i, line, err)
			break
		}
	}
}
