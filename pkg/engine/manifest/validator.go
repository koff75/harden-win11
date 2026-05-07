package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Validator wrap un schéma JSONSchema déjà compilé pour pouvoir valider
// plusieurs manifests sans recompiler le schéma à chaque appel.
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator compile le JSONSchema au chemin schemaPath. Retourne une
// erreur si le schéma est introuvable ou invalide.
func NewValidator(schemaPath string) (*Validator, error) {
	c := jsonschema.NewCompiler()
	s, err := c.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", schemaPath, err)
	}
	return &Validator{schema: s}, nil
}

// ValidateFile lit + parse + valide un manifest YAML contre le schéma compilé.
// Refuse les YAML multi-document (cohérent avec Load).
func (v *Validator) ValidateFile(manifestPath string) error {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var raw any
	dec := yaml.NewDecoder(bytes.NewReader(manifestData))
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return fmt.Errorf("manifest contains multiple YAML documents — split into separate files (one section per file)")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("parse YAML (extra docs): %w", err)
	}

	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("convert YAML→JSON: %w", err)
	}
	var instance any
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return fmt.Errorf("parse converted JSON: %w", err)
	}

	if err := v.schema.Validate(instance); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	// Vérification sémantique : rule.id doit être unique au sein de la section.
	// Le JSONSchema valide la structure mais pas l'unicité des items d'array.
	s, err := loadStructFromBytes(manifestData)
	if err != nil {
		return fmt.Errorf("re-parse for semantic check: %w", err)
	}
	seen := map[string]bool{}
	for _, r := range s.Rules {
		if seen[r.ID] {
			return fmt.Errorf("duplicate rule.id %q within section %q", r.ID, s.Section.ID)
		}
		seen[r.ID] = true
	}
	return nil
}

// loadStructFromBytes parse un manifest YAML déjà lu en bytes.
func loadStructFromBytes(data []byte) (*Section, error) {
	var s Section
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate est un raccourci qui compile le schéma + valide un manifest en
// une seule étape. Pour valider plusieurs manifests, préférer NewValidator
// + ValidateFile pour ne pas recompiler le schéma à chaque appel.
func Validate(manifestPath, schemaPath string) error {
	v, err := NewValidator(schemaPath)
	if err != nil {
		return err
	}
	return v.ValidateFile(manifestPath)
}
