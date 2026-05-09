package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/winadmin"
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

// TestCLI_Undo vérifie le flux undo end-to-end : on construit un journal
// fixture qui simule un apply réussi, on appelle 'harden-engine undo' avec
// --journal-dir et --manifest-dir pointant sur les fixtures, et on vérifie
// que les .undo.ps1 ont été lancés dans l'ordre inverse + que les rules
// irreversible sont skippées.
//
// Skip si non-admin (undo refuse de tourner sans privilèges).
func TestCLI_Undo(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	isAdmin, err := winadmin.IsElevated()
	if err != nil {
		t.Fatalf("admin check: %v", err)
	}
	if !isAdmin {
		t.Skip("undo requires admin (skipped on non-elevated CI/local)")
	}

	bin := buildEngine(t)
	root := repoRoot(t)
	tmpJournalDir := t.TempDir()

	// 1. Construire un journal fixture qui simule un run avec 3 rules :
	//    - undofixture.first  (applied avec before)
	//    - undofixture.second (applied avec before)
	//    - undofixture.irreversible (applied — undo doit la skip car irreversible)
	runID := "fixture-2026-05-08T00-00-00"
	events := []map[string]any{
		{"type": "run_start", "run_id": runID, "mode": "apply"},
		{"type": "section_start", "run_id": runID, "section_id": "undo_fixture"},
		{
			"type":       "action_result",
			"run_id":     runID,
			"section_id": "undo_fixture",
			"rule_id":    "undofixture.first",
			"status":     "applied",
			"before":     map[string]any{"value": "old1"},
			"after":      map[string]any{"value": "new1"},
		},
		{
			"type":       "action_result",
			"run_id":     runID,
			"section_id": "undo_fixture",
			"rule_id":    "undofixture.second",
			"status":     "applied",
			"before":     map[string]any{"value": "old2"},
			"after":      map[string]any{"value": "new2"},
		},
		{
			"type":       "action_result",
			"run_id":     runID,
			"section_id": "undo_fixture",
			"rule_id":    "undofixture.irreversible",
			"status":     "applied",
		},
		{"type": "section_end", "run_id": runID, "section_id": "undo_fixture"},
		{"type": "run_end", "run_id": runID, "applied": 3},
	}
	journalPath := filepath.Join(tmpJournalDir, runID+".ndjson")
	f, err := os.Create(journalPath)
	if err != nil {
		t.Fatalf("create journal: %v", err)
	}
	for _, ev := range events {
		b, _ := json.Marshal(ev)
		_, _ = f.Write(b)
		_, _ = f.Write([]byte("\n"))
	}
	_ = f.Close()

	// 2. Préparer un manifest-dir temporaire avec un manifest qui pointe vers
	//    les .undo.ps1 fixtures via des paths ABSOLUS — sinon BasePath
	//    (parent du manifest-dir) résoudrait les chemins relatifs hors du repo.
	tmpManifestDir := t.TempDir()
	manifest := buildAbsoluteUndoFixture(t, root)
	if err := os.WriteFile(filepath.Join(tmpManifestDir, "99-undo-fixture.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}

	// 3. Lancer 'harden-engine undo' sur le journal fixture.
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, "undo",
		"--run-id", runID,
		"--journal-dir", tmpJournalDir,
		"--manifest-dir", tmpManifestDir,
		"--yes",
	)
	cmd.Dir = root
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("undo command failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// 4. Parser la sortie NDJSON.
	var undoEvents []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line == "" {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("invalid JSON line: %q : %v", line, err)
			continue
		}
		undoEvents = append(undoEvents, ev)
	}

	// 5. Vérifier l'ordre LIFO : irreversible (skipped) → second (ok) → first (ok).
	var sawFirst, sawSecond, sawIrreversibleSkip bool
	var orderSecond, orderFirst int
	for i, ev := range undoEvents {
		if ev["type"] != "undo_result" {
			continue
		}
		switch ev["rule_id"] {
		case "undofixture.first":
			sawFirst = true
			orderFirst = i
			if ev["status"] != "ok" {
				t.Errorf("undofixture.first : expected status=ok, got %v (error: %v)", ev["status"], ev["error"])
			}
		case "undofixture.second":
			sawSecond = true
			orderSecond = i
			if ev["status"] != "ok" {
				t.Errorf("undofixture.second : expected status=ok, got %v (error: %v)", ev["status"], ev["error"])
			}
		case "undofixture.irreversible":
			sawIrreversibleSkip = true
			if ev["status"] != "skipped" {
				t.Errorf("undofixture.irreversible : expected status=skipped, got %v", ev["status"])
			}
		}
	}
	if !sawFirst {
		t.Error("did not see undo_result for undofixture.first")
	}
	if !sawSecond {
		t.Error("did not see undo_result for undofixture.second")
	}
	if !sawIrreversibleSkip {
		t.Error("did not see undo_result skip for undofixture.irreversible")
	}
	// LIFO : second doit être undo AVANT first (i.e. orderSecond < orderFirst).
	if sawFirst && sawSecond && orderSecond >= orderFirst {
		t.Errorf("expected LIFO order (second before first), got second@%d first@%d", orderSecond, orderFirst)
	}

	// 6. Vérifier qu'un journal undo-<id>.ndjson a été créé.
	undoJournalFiles, err := filepath.Glob(filepath.Join(tmpJournalDir, "undo-*.ndjson"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(undoJournalFiles) == 0 {
		t.Error("expected at least one undo-*.ndjson journal file")
	}
}

// TestCLI_Undo_NoRun vérifie qu'undo sur un dossier journal vide retourne
// exit 4 avec un message clair.
func TestCLI_Undo_NoRun(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows only")
	}
	bin := buildEngine(t)
	tmpJournalDir := t.TempDir()
	tmpManifestDir := t.TempDir()
	// Créer un manifest valide pour passer la phase de validation.
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "cmd", "harden-engine", "testdata", "undo-fixture.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpManifestDir, "99-fixture.yaml"), src, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command(bin, "undo",
		"--journal-dir", tmpJournalDir,
		"--manifest-dir", tmpManifestDir,
		"--yes",
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit, got success")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	// Exit 4 = input invalide (pas de journal trouvé).
	if exitErr.ExitCode() != 4 {
		t.Errorf("expected exit code 4, got %d (output: %s)", exitErr.ExitCode(), out)
	}
	if !strings.Contains(string(out), "no journal files") && !strings.Contains(string(out), "find latest run") {
		t.Errorf("expected error mentioning empty journal, got: %s", out)
	}
}

// buildAbsoluteUndoFixture construit le YAML d'un manifest fixture où les
// paths action/test/undo pointent vers des fichiers existants du repo, en
// chemins absolus YAML-safe (forward slashes pour éviter les soucis
// d'échappement Windows). Schema.json valide.
func buildAbsoluteUndoFixture(t *testing.T, root string) string {
	t.Helper()
	abs := func(rel string) string {
		p := filepath.Join(root, rel)
		return strings.ReplaceAll(p, `\`, `/`)
	}
	action := abs("pkg/engine/executor/testdata/success.action.ps1")
	test := abs("pkg/engine/executor/testdata/never_compliant.test.ps1")
	undo := abs("pkg/engine/executor/testdata/noop.undo.ps1")
	if _, err := os.Stat(filepath.FromSlash(undo)); err != nil {
		t.Fatalf("noop.undo.ps1 missing: %v", err)
	}
	// Le JSONSchema force action: '\.action\.ps1$' — nos paths absolus matchent.
	return fmt.Sprintf(`version: "1.0"

section:
  id: undo_fixture
  order: 99
  title: "Undo test fixture"
  description: "."

rules:
  - id: undofixture.first
    title: "First test rule"
    description: "."
    explanation: "."
    severity: nice-to-have
    impact: "."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    action: %q
    test: %q
    undo: %q

  - id: undofixture.second
    title: "Second test rule"
    description: "."
    explanation: "."
    severity: nice-to-have
    impact: "."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: false
    action: %q
    test: %q
    undo: %q

  - id: undofixture.irreversible
    title: "Irreversible rule"
    description: "."
    explanation: "."
    severity: nice-to-have
    impact: "."
    requires_reboot: false
    profile_when: always
    depends_on: []
    irreversible: true
    irreversible_reason: "Fixture pour vérifier skip undo."
    action: %q
    test: %q
`, action, test, undo, action, test, undo, action, test)
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
