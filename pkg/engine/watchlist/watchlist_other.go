//go:build !windows

package watchlist

import "syscall"

func hideConsoleAttrImpl() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
