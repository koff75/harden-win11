// Package restorepoint encapsule la création d'un Windows System Restore
// Point avant un apply réel. C'est la "ceinture-bretelles" sécurité : si tout
// part en sucette malgré le journal NDJSON et les .undo.ps1, l'utilisateur
// peut revenir à l'état OS-level via Recovery Environment.
//
// Fragilités connues :
//   - System Restore peut être désactivé sur Win11 par défaut. On essaie
//     de l'activer pour C: si ce n'est pas le cas (admin requis).
//   - Windows limite à 1 Restore Point créé par 24h via API standard.
//     Si l'OS en a déjà créé un (Windows Update du matin), notre Checkpoint
//     sera silencieusement skippé. On log l'event "skipped_throttled".
//   - L'API GetTokenInformation suffit pour vérifier l'élévation au moment
//     d'appeler ce helper (déjà fait par le caller via winadmin.IsElevated).
package restorepoint

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Status agrège le résultat d'une tentative de création.
type Status struct {
	Created     bool          `json:"created"`
	Description string        `json:"description"`
	Reason      string        `json:"reason,omitempty"` // si !Created : pourquoi
	Duration    time.Duration `json:"duration_ns"`
	Error       string        `json:"error,omitempty"` // détail technique en cas d'échec
}

// Create lance Checkpoint-Computer via PowerShell. Best-effort : ne renvoie
// jamais d'erreur fatale au caller — un échec doit être loggué et l'apply
// peut continuer (le journal NDJSON + .undo.ps1 reste la voie principale
// de rollback).
//
// timeout : durée max avant de tuer le PS (typiquement 60s, Checkpoint-Computer
// peut prendre 20-40s sur un disque chargé).
func Create(ctx context.Context, runID string, timeout time.Duration) Status {
	start := time.Now()
	desc := fmt.Sprintf("harden-win11 pre-apply %s", runID)

	// Le script :
	//   1. Tente d'activer System Restore sur C: si désactivé.
	//   2. Lance Checkpoint-Computer avec MODIFY_SETTINGS.
	//   3. Retourne JSON {ok, message}.
	// On capture proprement les WarningPreference (si throttling 24h, PS émet
	// un Warning sans erreur de retour, donc on doit checker la sortie).
	script := `
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

try {
    # 1. S'assurer que System Restore est actif sur C:.
    try {
        Enable-ComputerRestore -Drive 'C:\' -ErrorAction Stop
    } catch {}

    # 2. Capture les warnings (Windows envoie un warning si le 24h-throttle
    #    bloque la création) plutôt que de les perdre.
    $warnings = @()
    Checkpoint-Computer -Description '` + escapeSingleQuotes(desc) + `' ` + "`" + `
        -RestorePointType MODIFY_SETTINGS ` + "`" + `
        -WarningVariable warnings ` + "`" + `
        -WarningAction SilentlyContinue ` + "`" + `
        -ErrorAction Stop

    if ($warnings.Count -gt 0) {
        @{
            ok      = $false
            reason  = 'throttled'
            message = ($warnings | ForEach-Object { $_.Message }) -join ' | '
        } | ConvertTo-Json -Compress
    } else {
        @{ ok = $true; message = 'restore point created' } | ConvertTo-Json -Compress
    }
} catch {
    @{
        ok      = $false
        reason  = 'error'
        message = $_.Exception.Message
    } | ConvertTo-Json -Compress
}
`

	rctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(rctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = hideConsole()

	out, err := cmd.Output()
	st := Status{
		Description: desc,
		Duration:    time.Since(start),
	}
	if err != nil {
		// Le PS n'a même pas démarré ou a crashé hard.
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		st.Created = false
		st.Reason = "spawn_failed"
		st.Error = strings.TrimSpace(fmt.Sprintf("%v %s", err, stderr))
		return st
	}

	// Parser le JSON de retour. Sortie vide = créé silencieusement (rare).
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		st.Created = true
		return st
	}
	// On laisse le caller parser si besoin (le runner Go sait faire), mais ici
	// on fait simple via heuristique sur la string.
	if strings.Contains(trimmed, `"ok":true`) {
		st.Created = true
	} else {
		st.Created = false
		// Extrait le message via une regex simple-stupide.
		if i := strings.Index(trimmed, `"reason":"`); i >= 0 {
			rest := trimmed[i+10:]
			if j := strings.Index(rest, `"`); j > 0 {
				st.Reason = rest[:j]
			}
		}
		if i := strings.Index(trimmed, `"message":"`); i >= 0 {
			rest := trimmed[i+11:]
			if j := strings.Index(rest, `"`); j > 0 {
				st.Error = rest[:j]
			}
		}
	}
	return st
}

func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// hideConsole : variant Windows / non-Windows.
func hideConsole() *syscall.SysProcAttr {
	return hideConsoleImpl()
}
