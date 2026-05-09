//go:build !windows

package restorepoint

import "syscall"

// Stub non-Windows : SysProcAttr vide. Sur Linux/macOS le package n'a pas
// d'utilité, mais on doit compiler pour les CI cross-compile checks.
func hideConsoleImpl() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
