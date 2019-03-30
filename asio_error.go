package asio

import "syscall"

type errno uintptr

func STATUS_IS_SUCCESS(err error) bool {
	return err == nil || err == SUCCESS
}

func (e errno) IsAgain() bool {
	return syscall.Errno(e) == syscall.EAGAIN || syscall.Errno(e) == syscall.EWOULDBLOCK
}

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}
