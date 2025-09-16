//go:build linux
// +build linux

package backup

import (
	"syscall"
)

func Select(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) (n int, err error) {
	return syscall.Select(nfd, r, w, e, timeout)
}
