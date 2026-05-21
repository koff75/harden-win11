package executor

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolvePath_AcceptsRelativeUnderBase(t *testing.T) {
	base := t.TempDir()
	got, err := resolvePath(base, filepath.Join("engine", "actions", "defender", "realtime.test.ps1"))
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	wantPrefix, _ := filepath.Abs(base)
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("resolved path %q does not start with base %q", got, wantPrefix)
	}
}

func TestResolvePath_RejectsAbsolute(t *testing.T) {
	base := t.TempDir()
	var abs string
	if runtime.GOOS == "windows" {
		abs = `C:\Windows\System32\evil.ps1`
	} else {
		abs = "/etc/passwd"
	}
	if _, err := resolvePath(base, abs); err == nil {
		t.Fatalf("expected error for absolute path %q, got nil", abs)
	}
}

func TestResolvePath_RejectsTraversal(t *testing.T) {
	base := t.TempDir()
	cases := []string{
		filepath.Join("..", "evil.ps1"),
		filepath.Join("..", "..", "..", "..", "evil.ps1"),
		filepath.Join("engine", "..", "..", "evil.ps1"),
	}
	for _, c := range cases {
		if _, err := resolvePath(base, c); err == nil {
			t.Errorf("expected error for traversal path %q, got nil", c)
		}
	}
}

func TestResolvePath_RejectsUNC(t *testing.T) {
	base := t.TempDir()
	cases := []string{
		`\\attacker\share\evil.ps1`,
		`//attacker/share/evil.ps1`,
	}
	for _, c := range cases {
		if _, err := resolvePath(base, c); err == nil {
			t.Errorf("expected error for UNC path %q, got nil", c)
		}
	}
}

func TestResolvePath_RejectsEmpty(t *testing.T) {
	if _, err := resolvePath(t.TempDir(), ""); err == nil {
		t.Fatalf("expected error for empty path, got nil")
	}
}
