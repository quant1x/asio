// +build darwin dragonfly freebsd netbsd openbsd

package asio

import "syscall"

var (
	SO_REUSEPORT = syscall.SO_REUSEPORT
)
