package manifest

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load lit et parse un fichier manifest YAML.
//
// Le YAML doit respecter la structure définie par les types Section/SectionMeta/Rule.
// Cette fonction NE valide PAS contre le JSONSchema — utiliser Validate() pour ça.
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
	return &s, nil
}
