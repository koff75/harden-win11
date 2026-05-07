package runner

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRunner_RunPS_EchoFixture(t *testing.T) {
	r := New()
	scriptPath, _ := filepath.Abs("testdata/echo.ps1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := r.RunPS(ctx, scriptPath, map[string]any{"hello": "world", "n": 42})
	if err != nil {
		t.Fatalf("RunPS error: %v", err)
	}

	if out["hello"] != "world" {
		t.Errorf("expected hello=world, got %v", out["hello"])
	}
	if out["echoed"] != true {
		t.Errorf("expected echoed=true, got %v", out["echoed"])
	}
}

func TestRunner_RunPS_NoInput(t *testing.T) {
	r := New()
	scriptPath, _ := filepath.Abs("testdata/echo.ps1")

	ctx := context.Background()
	out, err := r.RunPS(ctx, scriptPath, nil)
	if err != nil {
		t.Fatalf("RunPS error: %v", err)
	}
	if out["echoed"] != true {
		t.Errorf("expected echoed=true, got %v", out["echoed"])
	}
}

func TestRunner_RunPS_MissingScript(t *testing.T) {
	r := New()
	ctx := context.Background()
	_, err := r.RunPS(ctx, "nonexistent.ps1", nil)
	if err == nil {
		t.Fatal("expected error for missing script, got nil")
	}
}
