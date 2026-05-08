// Package runner exécute des snippets PowerShell avec I/O JSON via stdin/stdout.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Runner exécute des scripts PowerShell.
type Runner struct {
	// Path vers powershell.exe. Vide = utilise "powershell.exe" (PATH).
	PowerShellPath string
}

// New retourne un Runner avec les défauts.
func New() *Runner {
	return &Runner{PowerShellPath: "powershell.exe"}
}

// RunPS exécute le script PS au chemin scriptPath en lui passant input
// sérialisé en JSON sur stdin. Retourne le JSON parsé depuis stdout.
//
// Le script doit lire stdin via [Console]::In.ReadToEnd() et émettre du JSON
// en une ligne sur stdout (via ConvertTo-Json -Compress).
//
// stderr du process est capturé et inclus dans l'erreur si le process échoue.
func (r *Runner) RunPS(ctx context.Context, scriptPath string, input any) (map[string]any, error) {
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("script not found: %w", err)
	}

	cmd := exec.CommandContext(ctx, r.PowerShellPath,
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	)
	hideConsoleWindow(cmd)

	// Sérialise input en JSON (ou string vide si nil)
	var stdin []byte
	if input != nil {
		var err error
		stdin, err = json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal input: %w", err)
		}
	}
	cmd.Stdin = bytes.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell failed: %w (stderr: %s)", err, stderr.String())
	}

	out := stdout.Bytes()
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, fmt.Errorf("powershell produced empty stdout (stderr: %s)", stderr.String())
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse stdout as JSON: %w (stdout: %s)", err, string(out))
	}
	return result, nil
}
