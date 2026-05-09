package journal

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzReadRun : envoie du NDJSON malformé à ReadRun() pour vérifier qu'il
// n'y a pas de panic (le journal est consommé par la GUI au boot, un crash
// = écran blanc).
func FuzzReadRun(f *testing.F) {
	// Seeds : NDJSON valides + edge cases.
	f.Add([]byte(`{"type":"run_start","run_id":"r1"}
{"type":"action_result","rule_id":"defender.realtime","status":"applied"}
{"type":"run_end","run_id":"r1"}
`))
	f.Add([]byte(""))
	f.Add([]byte("not json\n"))
	f.Add([]byte(`{"type":"run_start"}` + "\n" + `{this is broken}` + "\n"))
	f.Add([]byte(`{"a":` + string(make([]byte, 1000)) + `}`))
	// Lignes très longues (memory pressure).
	long := append([]byte(`{"k":"`), make([]byte, 10000)...)
	long = append(long, []byte(`"}`+"\n")...)
	f.Add(long)
	// Mix de \r\n / \n / \r.
	f.Add([]byte("{\"a\":1}\r\n{\"b\":2}\n{\"c\":3}\r"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		runID := "fuzz-test"
		path := filepath.Join(dir, runID+".ndjson")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Skip("write failed")
		}
		// ReadRun ne doit JAMAIS panic.
		_, _ = ReadRun(dir, runID)
	})
}
