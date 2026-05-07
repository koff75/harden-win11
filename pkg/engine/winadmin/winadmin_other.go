//go:build !windows

package winadmin

// isElevatedWindows est un stub pour les OS non-Windows (le moteur est
// Windows-only mais le code Go doit compiler partout pour CI/dev).
func isElevatedWindows() (bool, error) {
	return false, nil
}
