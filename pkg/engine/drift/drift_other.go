//go:build !windows

package drift

import "syscall"

func hideConsoleAttrImpl() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
