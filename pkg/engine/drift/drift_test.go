package drift

import (
	"path/filepath"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/snapshot"
)

func TestSameValue(t *testing.T) {
	a := snapshot.RegEntry{Exists: true, Value: 1}
	b := snapshot.RegEntry{Exists: true, Value: 1}
	if !sameValue(a, b) {
		t.Error("identical entries should be equal")
	}
	c := snapshot.RegEntry{Exists: true, Value: 2}
	if sameValue(a, c) {
		t.Error("different values should differ")
	}
	d := snapshot.RegEntry{Exists: false}
	if sameValue(a, d) {
		t.Error("exists/not-exists should differ")
	}
}

func TestDefaultDir(t *testing.T) {
	dir := DefaultDir()
	if dir == "" {
		t.Error("DefaultDir should not be empty")
	}
	if filepath.Base(dir) != "drift" {
		t.Errorf("expected dir name 'drift', got %q", filepath.Base(dir))
	}
}
