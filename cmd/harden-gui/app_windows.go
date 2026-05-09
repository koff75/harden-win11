//go:build windows

package main

import "syscall"

// hideWindowAttr renvoie un SysProcAttr qui empêche powershell.exe de flasher
// une console visible quand on spawn un sous-process depuis la GUI.
// CREATE_NO_WINDOW = 0x08000000.
func hideWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
