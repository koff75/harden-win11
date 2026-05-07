package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load lit et parse un fichier manifest YAML.
//
// Politique : 1 fichier = 1 section. Un YAML multi-document (séparateurs '---')
// est refusé pour éviter qu'un 2e document soit silencieusement ignoré.
//
// KnownFields(true) force l'échec sur tout champ YAML non mappé dans les types,
// ce qui détecte les fautes de frappe (ex: 'severty' au lieu de 'severity').
func Load(path string) (*Section, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}

	var s Section
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", path, err)
	}

	// Refuse les YAML multi-document : si un 2e Decode réussit, il y a
	// un 2e document après le séparateur '---' qui serait sinon ignoré.
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("manifest %s contains multiple YAML documents — split into separate files (one section per file)", path)
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse manifest %s (extra docs): %w", path, err)
	}

	return &s, nil
}
