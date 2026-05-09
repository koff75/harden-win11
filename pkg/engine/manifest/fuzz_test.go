package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad : envoie des YAML malformés à manifest.Load() pour vérifier qu'il
// n'y a pas de crash (panic), juste des erreurs propres.
//
// Lancer : go test -fuzz=FuzzLoad -fuzztime=30s ./pkg/engine/manifest/
//
// Le seed corpus inclut un YAML valide minimal + des cas connus (vide,
// bytes random, YAML profond, etc).
func FuzzLoad(f *testing.F) {
	// Seed corpus : YAML valides + edge cases.
	f.Add([]byte(`version: "1.0"
section:
  id: foo
  order: 1
  title: "Foo"
  description: "Test"
rules: []
`))
	f.Add([]byte(""))
	f.Add([]byte("---\n---\n"))
	f.Add([]byte("not yaml at all"))
	f.Add([]byte("key: !!python/object\n"))
	// YAML très imbriqué (potentiel stack overflow avec mauvais parser).
	deep := []byte("a:\n")
	for i := 0; i < 50; i++ {
		deep = append(deep, []byte("  b:\n")...)
	}
	f.Add(deep)
	// Champ non-attendu (le loader strict doit refuser).
	f.Add([]byte(`version: "1.0"
unknown_field: "should be rejected"
section:
  id: x
  order: 1
  title: "T"
  description: "D"
rules: []
`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Écrit dans un fichier temp puis appelle Load.
		tmp, err := os.CreateTemp(t.TempDir(), "fuzz-*.yaml")
		if err != nil {
			t.Skip("cannot create temp file")
		}
		defer tmp.Close()
		if _, err := tmp.Write(data); err != nil {
			t.Skip("write failed")
		}
		path := tmp.Name()
		_ = filepath.Clean(path)
		// Important : Load NE DOIT PAS panic, même sur YAML aberrant.
		// Une error retournée est un comportement OK.
		_, _ = Load(path)
	})
}
