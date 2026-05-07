package manifest

import (
	"path/filepath"
	"testing"
)

func TestValidate_ValidManifest(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/valid-defender.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	if err := Validate(manifestPath, schemaPath); err != nil {
		t.Fatalf("Validate returned error on valid manifest: %v", err)
	}
}

func TestValidate_MissingRequiredField(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/missing-action.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	err := Validate(manifestPath, schemaPath)
	if err == nil {
		t.Fatal("expected validation error for missing 'action', got nil")
	}
}

func TestValidate_InvalidSeverity(t *testing.T) {
	manifestPath, _ := filepath.Abs("testdata/invalid-severity.yaml")
	schemaPath, _ := filepath.Abs("../../../schemas/manifest.schema.json")

	err := Validate(manifestPath, schemaPath)
	if err == nil {
		t.Fatal("expected validation error for invalid severity, got nil")
	}
}
