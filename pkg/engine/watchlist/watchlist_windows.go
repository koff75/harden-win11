//go:build windows

package watchlist

import "syscall"

func hideConsoleAttrImpl() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
