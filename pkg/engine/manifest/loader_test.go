package manifest

import (
	"path/filepath"
	"testing"
)

func TestLoad_ValidDefender(t *testing.T) {
	path, _ := filepath.Abs("testdata/valid-defender.yaml")

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if s.Version != "1.0" {
		t.Errorf("expected version 1.0, got %q", s.Version)
	}
	if s.Section.ID != "defender" {
		t.Errorf("expected section.id=defender, got %q", s.Section.ID)
	}
	if s.Section.Order != 1 {
		t.Errorf("expected section.order=1, got %d", s.Section.Order)
	}
	if len(s.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(s.Rules))
	}

	r := s.Rules[0]
	if r.ID != "defender.realtime" {
		t.Errorf("expected rule.id=defender.realtime, got %q", r.ID)
	}
	if r.Severity != "critical" {
		t.Errorf("expected severity=critical, got %q", r.Severity)
	}
	if r.ProfileWhen != "always" {
		t.Errorf("expected profile_when=always, got %q", r.ProfileWhen)
	}
	if r.Action != "./engine/actions/defender/realtime.action.ps1" {
		t.Errorf("unexpected action path: %q", r.Action)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path, _ := filepath.Abs("testdata/invalid-syntax.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
