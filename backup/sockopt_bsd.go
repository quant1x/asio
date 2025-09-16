//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

package backup

import "syscall"

var (
	SO_REUSEPORT = syscall.SO_REUSEPORT
)
