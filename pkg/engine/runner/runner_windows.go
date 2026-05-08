//go:build windows

package runner

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow empêche powershell.exe d'ouvrir une fenêtre console
// visible (le flash gris/noir agaçant en GUI). CREATE_NO_WINDOW = 0x08000000.
//
// Sans ce flag, le runner Go spawn un powershell.exe qui crée une console
// même si stdin/stdout/stderr sont redirigés. Avec CREATE_NO_WINDOW + HideWindow,
// le process tourne complètement en arrière-plan.
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
