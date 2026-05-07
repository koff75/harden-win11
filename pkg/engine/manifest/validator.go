package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Validate vérifie qu'un manifest YAML respecte le schéma JSONSchema fourni.
//
// Le YAML est d'abord parsé en `any`, puis converti en JSON, puis validé
// (le validateur JSONSchema ne lit pas YAML directement).
func Validate(manifestPath, schemaPath string) error {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var raw any
	if err := yaml.Unmarshal(manifestData, &raw); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("convert YAML→JSON: %w", err)
	}
	var instance any
	if err := json.Unmarshal(jsonBytes, &instance); err != nil {
		return fmt.Errorf("parse converted JSON: %w", err)
	}

	c := jsonschema.NewCompiler()
	schema, err := c.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}
	return nil
}
