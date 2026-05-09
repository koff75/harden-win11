//go:build windows

package executor

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koff75/harden-win11/pkg/engine/ndjson"
	"github.com/koff75/harden-win11/pkg/engine/runner"
)

// TestE2E_HKCU_ApplyThenVerify : vrai test E2E. Apply une rule qui set une
// valeur dans HKCU\Software\HardenWin11Test\, vérifie que le test la voit
// compliant après, puis cleanup.
//
// Ne nécessite pas admin (HKCU = current user). Pas de side-effect global.
//
// Couvre la chaîne complète :
//   - manifest.Load
//   - executor.Run en ModeApply
//   - .test.ps1 → non-compliant
//   - .action.ps1 → écrit la regkey
//   - re-test post-apply → compliant
//   - status=applied
func TestE2E_HKCU_ApplyThenVerify(t *testing.T) {
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-hkcu-e2e.yaml")

	// Cleanup avant : assure qu'on part d'un état non-compliant.
	cleanupHKCU(t)
	defer cleanupHKCU(t)

	var buf bytes.Buffer
	w := ndjson.NewWriter(&buf)
	summary, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   w,
		RunID:    "test-e2e-hkcu",
	})
	if err != nil {
		t.Fatalf("Run apply: %v", err)
	}
	if summary.Applied != 1 {
		t.Errorf("expected Applied=1, got %+v", summary)
	}

	// Vérification REELLE via PS : la regkey doit exister maintenant.
	if !regKeyHas(t, "HKCU:\\Software\\HardenWin11Test", "TestE2EValue", "42") {
		t.Error("apply n'a pas créé la regkey HKCU:\\Software\\HardenWin11Test\\TestE2EValue=42")
	}

	// Lance un dryrun maintenant : la rule doit être would_skip (compliant).
	var buf2 bytes.Buffer
	w2 := ndjson.NewWriter(&buf2)
	summary2, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeDry,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   w2,
		RunID:    "test-e2e-hkcu-recheck",
	})
	if err != nil {
		t.Fatalf("Run dryrun: %v", err)
	}
	if summary2.Skipped != 1 {
		t.Errorf("expected Skipped=1 after apply (rule should now be compliant), got %+v", summary2)
	}

	// Vérifie que l'event applied contient le 'recheck=compliant' (testé par
	// le post-apply re-test).
	events := parseEvents(t, buf.Bytes())
	var sawRecheck bool
	for _, ev := range events {
		if ev["rule_id"] == "hkcu_sandbox.test_value" && ev["recheck"] == "compliant" {
			sawRecheck = true
			break
		}
	}
	if !sawRecheck {
		t.Error("expected recheck=compliant on the applied event")
	}
}

// TestE2E_HKCU_FullCycle : apply puis undo, vérifie retour à l'état initial.
func TestE2E_HKCU_FullCycle(t *testing.T) {
	repo := mustFindRepo(t)
	manifestPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "fixture-hkcu-e2e.yaml")

	cleanupHKCU(t)
	defer cleanupHKCU(t)

	// 1. Apply
	var applyBuf bytes.Buffer
	_, err := Run(context.Background(), manifestPath, Options{
		Mode:     ModeApply,
		BasePath: repo,
		Runner:   runner.New(),
		Writer:   ndjson.NewWriter(&applyBuf),
		RunID:    "e2e-cycle-apply",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !regKeyHas(t, "HKCU:\\Software\\HardenWin11Test", "TestE2EValue", "42") {
		t.Fatal("post-apply : regkey absente")
	}

	// 2. Undo en lisant le before depuis l'event applied.
	events := parseEvents(t, applyBuf.Bytes())
	var beforeState any
	for _, ev := range events {
		if ev["rule_id"] == "hkcu_sandbox.test_value" && ev["status"] == "applied" {
			beforeState = ev["before"]
		}
	}
	if beforeState == nil {
		t.Fatal("no before state captured in apply event")
	}

	// Lance directement le undo.ps1 avec le before en stdin.
	r := runner.New()
	undoPath := filepath.Join(repo, "pkg", "engine", "executor", "testdata", "hkcu-sandbox.undo.ps1")
	out, err := r.RunPS(context.Background(), undoPath, beforeState)
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if ok, _ := out["ok"].(bool); !ok {
		t.Errorf("undo did not return ok=true: %+v", out)
	}

	// 3. Vérifie que la regkey est revenue à son état initial (absente).
	if regKeyHas(t, "HKCU:\\Software\\HardenWin11Test", "TestE2EValue", "42") {
		t.Error("undo n'a pas restauré l'état initial : regkey toujours à 42")
	}
}

func cleanupHKCU(t *testing.T) {
	t.Helper()
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		`Remove-ItemProperty -Path 'HKCU:\Software\HardenWin11Test' -Name 'TestE2EValue' -ErrorAction SilentlyContinue`)
	_ = cmd.Run()
}

func regKeyHas(t *testing.T, path, name, expected string) bool {
	t.Helper()
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command",
		`try { (Get-ItemProperty -Path '`+path+`' -Name '`+name+`' -ErrorAction Stop).`+name+` } catch { '' }`)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out)) == expected
}
