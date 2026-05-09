//go:build !windows

package snapshot

import "syscall"

func hideConsoleAttrImpl() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
