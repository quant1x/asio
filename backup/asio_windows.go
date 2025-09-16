//go:build windows
// +build windows

package backup

import "syscall"

type socket_t = syscall.Handle
