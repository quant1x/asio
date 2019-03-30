package os

/*
#include <errno.h>
#include <stdio.h>
#include <sys/socket.h>
 */
import "C"

import (
	"fmt"
	"syscall"
	"unsafe"
)

func Socket(domain, typ, proto int) (fd int, err error) {
	return syscall.Socket(domain, typ, proto)
}

func Close(fd int) (error) {
	return syscall.Close(fd)
}

func Accept(fd int) (int, syscall.Sockaddr, error) {
	return syscall.Accept(fd)
}

func Connect(fd int, sa syscall.Sockaddr) (error) {
	return syscall.Connect(fd, sa)
}

func Recv(fd int, data []byte) (int, error) {
	return syscall.Read(fd, data)
}

func RawRecv(fd int, data []byte) (int, error) {
	var length C.size_t
	length = C.size_t(len(data))
	n, err := C.recv(C.int(fd), unsafe.Pointer(&data[0]), length, 0)
	if n < 0 {
		fmt.Println(err)
	}

	return int(n), err
}

func Send(fd int, data []byte) (int, error) {
	return syscall.Write(fd, data)
}

func RawSend(fd int, data []byte) (int, error) {
	var length C.size_t
	length = C.size_t(len(data))
	n, err := C.send(C.int(fd), unsafe.Pointer(&data[0]), length, 0)
	if n < 0 {
		fmt.Println(err)
	}

	return int(n), err
}
