//go:build !windows

package runner

import "os/exec"

// hideConsoleWindow est un no-op sur les OS non-Windows (le moteur est
// Windows-only mais le code Go doit compiler partout pour CI/dev).
func hideConsoleWindow(*exec.Cmd) {}
