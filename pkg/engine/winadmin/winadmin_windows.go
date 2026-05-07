//go:build windows

package winadmin

import (
	"os"
	"path/filepath"
)

// isElevatedWindows détecte un process élevé en tentant d'ouvrir en write un
// fichier dans %SystemRoot%\Temp\ (= C:\Windows\Temp\). Ce dossier accepte les
// writes uniquement pour les admins. Si la création réussit, on est admin.
//
// Cette approche est délibérément low-tech (pas de SYSCALL, pas de dépendance
// externe). Pour une détection plus robuste, on pourrait utiliser
// windows.GetCurrentProcessToken + IsElevated, mais on resterait en pratique
// avec le même résultat et plus de complexité.
func isElevatedWindows() (bool, error) {
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	probe := filepath.Join(sysRoot, "Temp", ".harden-engine-admin-probe")
	f, err := os.Create(probe)
	if err != nil {
		return false, nil
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return true, nil
}
