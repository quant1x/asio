package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

var (
	modWs2_32    = syscall.NewLazyDLL("ws2_32.dll")
	proConnectEx = modWs2_32.NewProc("ConnectEx")
	modKernel32  = syscall.NewLazyDLL("kernel32.dll")
)

func main() {
	// 创建TCP套接字
	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(socket)

	// 设置为非阻塞模式
	if err := windows.SetHandleInformation(windows.Handle(socket), windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		panic(err)
	}
	var mode uint32 = 1
	if code := syscall.SetHandleInformation(syscall.Handle(socket), syscall.HANDLE_FLAG_INHERIT, mode); code != nil {
		panic(code)
	}

	// 绑定本地地址（ConnectEx要求）
	localAddr := &syscall.SockaddrInet4{Port: 0}
	if err := syscall.Bind(socket, localAddr); err != nil {
		panic(err)
	}

	// 创建IOCP
	iocp, err := windows.CreateIoCompletionPort(windows.Handle(socket), 0, 0, 0)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(iocp)

	// 启动事件循环
	go eventLoop(iocp)

	// 异步连接
	targetAddr := &syscall.SockaddrInet4{Port: 8080, Addr: [...]byte{127, 0, 0, 1}}
	sa, _ := targetAddr.Sockaddr()
	overlapped := &windows.Overlapped{}
	_, _, errno := syscall.SyscallN(
		proConnectEx.Addr(),
		uintptr(socket),
		uintptr(unsafe.Pointer(&sa.(*syscall.RawSockaddrAny).Addr[0])),
		uintptr(len(sa)),
		0, 0, uintptr(unsafe.Pointer(overlapped)),
	)
	if errno != 0 {
		if errno != syscall.ERROR_IO_PENDING {
			panic(errno)
		}
	}

	// 等待连接完成（实际应在事件循环处理）
	select {}
}

func eventLoop(iocp windows.Handle) {
	for {
		var bytesTransferred uint32
		var key uintptr
		var overlapped *windows.Overlapped
		err := windows.GetQueuedCompletionStatus(iocp, &bytesTransferred, &key, &overlapped, windows.INFINITE)
		if err != nil {
			fmt.Println("IOCP错误:", err)
			continue
		}
		fmt.Println("操作完成，传输字节数:", bytesTransferred)
	}
}
