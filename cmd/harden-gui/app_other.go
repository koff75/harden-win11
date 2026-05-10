//go:build !windows

package main

import "syscall"

// Stub non-Windows : la GUI Wails ne tourne que sous Windows, mais on permet
// au cross-compile linux de réussir (CI sanity check). Sur les OS qui n'ont
// pas le concept "hide window for child process", on retourne nil.
func hideWindowAttr() *syscall.SysProcAttr {
	return nil
}
