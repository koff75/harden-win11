package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot remonte jusqu'à la racine du repo (où vit go.mod).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// cmd/harden-engine/main_test.go → racine = ../..
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// buildEngine compile le binaire dans dist/ et retourne son chemin absolu.
func buildEngine(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	out := filepath.Join(root, "dist", "harden-engine.exe")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/harden-engine")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, output)
	}
	return out
}

func TestCLI_Version(t *testing.T) {
	bin := buildEngine(t)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("version command failed: %v (stderr: %s)", err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("expected JSON output, got %q : %v", out, err)
	}
	if _, ok := v["version"]; !ok {
		t.Errorf("expected 'version' field in output, got %v", v)
	}
}
