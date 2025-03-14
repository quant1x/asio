package os

import (
	"fmt"
	"syscall"
	"unsafe"
)

func Socket(domain, typ, proto int) (fd int, err error) {
	return syscall.Socket(domain, typ, proto)
}

func Close(fd int) error {
	return syscall.Close(fd)
}

func Accept(fd int) (int, syscall.Sockaddr, error) {
	return syscall.Accept(fd)
}

func Connect(fd int, sa syscall.Sockaddr) error {
	return syscall.Connect(fd, sa)
}

func Recv(fd int, data []byte) (int, error) {
	return syscall.Read(fd, data)
}

func Send(fd int, data []byte) (int, error) {
	return syscall.Write(fd, data)
}
