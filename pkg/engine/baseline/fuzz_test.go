package baseline

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad : envoie des YAML mappings/ malformés au loader baseline.
func FuzzLoad(f *testing.F) {
	f.Add([]byte(`version: "1.0"
title: "Test"
sources: {}
disclaimer: "Hello"
mappings:
  defender.realtime:
    cis: ["18.10.42.12.1"]
    anssi: []
    ms_baseline: []
`))
	f.Add([]byte(""))
	f.Add([]byte("garbage"))
	f.Add([]byte(`mappings:
  rule.x:
    cis: null
    anssi: ["R30"]
    ms_baseline:
`))
	f.Add([]byte(`mappings:
  rule.invalid_yaml: { cis: [`))

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "fuzz-baseline.yaml")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Skip()
		}
		_, _ = Load(path)
	})
}
