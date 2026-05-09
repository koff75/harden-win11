package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// TestRunner_ConcurrentRunPS_NoRace : le Runner est utilisé par le pool de
// workers de runDryParallel — vérifions que des appels concurrents ne se
// marchent pas dessus (cmd, env, stdin, stdout sont alloués par-call).
//
// À lancer avec `go test -race ./pkg/engine/runner/` (nécessite CGO/gcc).
// Sans -race, on vérifie au moins que les outputs sont indépendants.
func TestRunner_ConcurrentRunPS_NoRace(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only (PowerShell)")
	}
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "echo.ps1")
	if err := os.WriteFile(scriptPath, []byte(`
$ErrorActionPreference = 'Stop'
$json = [Console]::In.ReadToEnd()
$obj = $json | ConvertFrom-Json
@{ id = $obj.id; ok = $true } | ConvertTo-Json -Compress
`), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	r := New()
	const concurrent = 8
	const callsPerGoroutine = 3
	var wg sync.WaitGroup
	errCh := make(chan error, concurrent*callsPerGoroutine)

	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < callsPerGoroutine; j++ {
				expected := float64(workerID*100 + j)
				out, err := r.RunPS(ctx, scriptPath, map[string]any{"id": expected})
				if err != nil {
					errCh <- err
					return
				}
				gotID, _ := out["id"].(float64)
				if gotID != expected {
					errCh <- &mismatchErr{expected: expected, got: gotID, worker: workerID}
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent error: %v", err)
	}
}

type mismatchErr struct {
	expected float64
	got      float64
	worker   int
}

func (e *mismatchErr) Error() string {
	return formatMismatch(e)
}

func formatMismatch(e *mismatchErr) string {
	// pas de fmt.Sprintf import déjà présent — fait le formatting manuel
	return "worker " + itoa(e.worker) + " expected " + ftoa(e.expected) + " got " + ftoa(e.got)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	out := ""
	for i > 0 {
		out = string(rune('0'+i%10)) + out
		i /= 10
	}
	return out
}

func ftoa(f float64) string {
	return itoa(int(f))
}
