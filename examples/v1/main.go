package main

import (
	"errors"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	AF_INET     = windows.AF_INET
	SOCK_STREAM = windows.SOCK_STREAM
	IPPROTO_TCP = windows.IPPROTO_TCP
)

var (
	modws2_32        = windows.NewLazySystemDLL("ws2_32.dll")
	proconnectex     = modws2_32.NewProc("ConnectEx")
	connectExFuncPtr uintptr
)

func main() {
	// 初始化Winsock
	var wsaData windows.WSAData
	if err := windows.WSAStartup(2<<16|2, &wsaData); err != nil {
		log.Fatal("WSAStartup failed:", err)
	}
	defer windows.WSACleanup()

	// 创建IOCP
	iocp, err := windows.CreateIoCompletionPort(windows.InvalidHandle, 0, 0, 0)
	if err != nil {
		log.Fatal("CreateIoCompletionPort failed:", err)
	}
	defer windows.CloseHandle(iocp)

	// 创建socket
	socket, err := windows.WSASocket(AF_INET, SOCK_STREAM, IPPROTO_TCP, nil, 0, windows.WSA_FLAG_OVERLAPPED)
	if err != nil {
		log.Fatal("WSASocket failed:", err)
	}
	defer windows.CloseHandle(socket)
	//// 获取ConnectEx函数指针
	//err = getConnectExFunc(syscall.Handle(socket))
	//if err != nil {
	//	panic(err)
	//}
	// 将socket关联到IOCP
	if _, err := windows.CreateIoCompletionPort(socket, iocp, 0, 0); err != nil {
		log.Fatal("CreateIoCompletionPort failed:", err)
	}

	// 准备连接地址
	// 目标地址
	sa := sockaddrInet{
		Family: AF_INET,
		Port:   uint16(8080),
		Addr:   [4]byte{127, 0, 0, 1},
	}
	*(*uint16)(unsafe.Pointer(&sa.Port)) = htons(sa.Port)
	// 异步连接
	overlapped := &syscall.Overlapped{}
	//r1, _, err := proconnectex.Call(
	//	uintptr(socket),
	//	uintptr(unsafe.Pointer(sa)),
	//	uintptr(unsafe.Sizeof(*sa)),
	//	0,
	//	0,
	//	0,
	//	uintptr(unsafe.Pointer(&overlapped)),
	//)
	err = connectEx(syscall.Handle(socket), sa, overlapped)

	if err != nil && err != syscall.ERROR_IO_PENDING {
		panic(err)
	}

	// 处理完成端口事件
	go func() {
		var bytesTransferred uint32
		var completionKey uintptr
		var overlapped *windows.Overlapped

		for {
			err := windows.GetQueuedCompletionStatus(iocp, &bytesTransferred, &completionKey, &overlapped, windows.INFINITE)
			if err != nil {
				if err == syscall.Errno(windows.WAIT_TIMEOUT) {
					continue
				}
				log.Println("GetQueuedCompletionStatus error:", err)
				return
			}

			if overlapped == nil {
				log.Println("Received shutdown signal")
				return
			}

			// 处理连接完成
			log.Printf("Operation completed, bytes transferred: %d", bytesTransferred)

			// 这里可以添加发送/接收数据的逻辑
		}
	}()

	// 示例：发送数据
	sendBuf := []byte("GET / HTTP/1.0\r\n\r\n")
	var sendOverlapped windows.Overlapped
	var sendBufs []windows.WSABuf
	sendBufs = append(sendBufs, windows.WSABuf{
		Len: uint32(len(sendBuf)),
		Buf: &sendBuf[0],
	})

	var flags uint32
	err = windows.WSASend(socket, &sendBufs[0], 1, nil, flags, &sendOverlapped, nil)
	if err != nil && err != syscall.ERROR_IO_PENDING {
		log.Fatal("WSASend failed:", err)
	}

	// 保持程序运行
	runtime.KeepAlive(sa)
	select {}
}

// 手动实现htons
func htons(n uint16) uint16 {
	return (n << 8) | (n >> 8)
}

func connectEx(s syscall.Handle, addr sockaddrInet, overlapped *syscall.Overlapped) error {
	err := LoadConnectEx()
	if err != nil {
		return errors.New("failed to find ConnectEx: " + err.Error())
	}
	addrBuf, addrLen, err := addr.sockaddr()
	if err != nil {
		return err
	}

	//r, _, err := syscall.Syscall6(
	//	connectExFuncPtr,
	//	6,
	//	uintptr(s),
	//	uintptr(addrBuf),
	//	uintptr(addrLen),
	//	0,
	//	0,
	//	uintptr(unsafe.Pointer(overlapped)),
	//)
	//if r == 0 {
	//	if !errors.Is(err, syscall.ERROR_IO_PENDING) {
	//		log.Fatal("ConnectEx failed:", err)
	//	} else {
	//		return err
	//	}
	//}
	r1, _, errno := syscall.Syscall9(
		connectExFunc.addr,
		7,
		uintptr(s),
		uintptr(addrBuf),
		uintptr(addrLen),
		0, //uintptr(unsafe.Pointer(sendBuf)),
		0, //uintptr(sendDataLen),
		0, //uintptr(unsafe.Pointer(bytesSent)),
		uintptr(unsafe.Pointer(overlapped)),
		0,
		0)
	if r1 == 0 {
		if errno != 0 {
			err = error(errno)
		} else {
			err = syscall.EINVAL
		}
	}
	return nil
}

type sockaddrInet struct {
	Family uint16
	Port   uint16
	Addr   [4]byte
	Zero   [8]byte
}

func (sa *sockaddrInet) sockaddr() (unsafe.Pointer, int32, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, syscall.EINVAL
	}
	port := sa.Port
	p := (*[2]byte)(unsafe.Pointer(&sa.Port))
	p[0] = byte(port >> 8)
	p[1] = byte(port)
	return unsafe.Pointer(sa), int32(unsafe.Sizeof(*sa)), nil
}

var connectExFunc struct {
	once sync.Once
	addr uintptr
	err  error
}

func LoadConnectEx() error {
	connectExFunc.once.Do(func() {
		var s syscall.Handle
		s, connectExFunc.err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
		if connectExFunc.err != nil {
			return
		}
		defer syscall.CloseHandle(s)
		var n uint32
		connectExFunc.err = syscall.WSAIoctl(s,
			syscall.SIO_GET_EXTENSION_FUNCTION_POINTER,
			(*byte)(unsafe.Pointer(&syscall.WSAID_CONNECTEX)),
			uint32(unsafe.Sizeof(syscall.WSAID_CONNECTEX)),
			(*byte)(unsafe.Pointer(&connectExFunc.addr)),
			uint32(unsafe.Sizeof(connectExFunc.addr)),
			&n, nil, 0)
	})
	return connectExFunc.err
}
