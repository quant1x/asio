// +build darwin freebsd dragonfly netbsd openbsd

package asio

import (
	"syscall"
)

func Select(nfd int, r *syscall.FdSet, w *syscall.FdSet, e *syscall.FdSet, timeout *syscall.Timeval) (n int, err error) {
	err = syscall.Select(nfd, r, w, e, timeout)
	return 0, err
}
