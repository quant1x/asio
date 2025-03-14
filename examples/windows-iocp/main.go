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
	//// 创建TCP套接字
	//socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	//if err != nil {
	//	panic(err)
	//}
	//defer syscall.Close(socket)
	// 设置为非阻塞模式
	var mode uint32 = 1
	if err := windows.SetHandleInformation(windows.Handle(socket), windows.HANDLE_FLAG_INHERIT, mode); err != nil {
		panic(err)
	}
	//var mode uint32 = 1
	//if code := syscall.SetHandleInformation(syscall.Handle(socket), syscall.HANDLE_FLAG_INHERIT, mode); code != nil {
	//	panic(code)
	//}
	// 绑定本地地址（ConnectEx要求）
	localAddr := &windows.SockaddrInet4{Port: 0}
	if err := windows.Bind(socket, localAddr); err != nil {
		panic(err)
	}
	// 将socket关联到IOCP
	_iocp, err := windows.CreateIoCompletionPort(socket, iocp, 0, 0)
	if err != nil {
		log.Fatal("CreateIoCompletionPort failed:", err)
	}
	defer windows.CloseHandle(_iocp)

	// 目标地址
	//sa := sockaddrInet{
	//	Family: AF_INET,
	//	Port:   uint16(8080),
	//	Addr:   [4]byte{127, 0, 0, 1},
	//}
	//*(*uint16)(unsafe.Pointer(&sa.Port)) = htons(sa.Port)
	// 异步连接
	//overlapped := &syscall.Overlapped{}
	//err = ConnectEx(syscall.Handle(socket), sa, nil, 0, nil, overlapped)
	overlapped := &windows.Overlapped{}
	sa := windows.SockaddrInet4{
		Port: 8080,
		Addr: [4]byte{127, 0, 0, 1},
	}
	err = windows.ConnectEx(socket, &sa, nil, 0, nil, overlapped)
	if err != nil && !errors.Is(err, syscall.ERROR_IO_PENDING) {
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
				if errors.Is(err, windows.WAIT_TIMEOUT) {
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
	err = windows.WSASend(windows.Handle(socket), &sendBufs[0], 1, nil, flags, &sendOverlapped, nil)
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

func connectEx(s syscall.Handle, name unsafe.Pointer, namelen int32, sendBuf *byte, sendDataLen uint32, bytesSent *uint32, overlapped *syscall.Overlapped) (err error) {
	r1, _, e1 := syscall.Syscall9(
		connectExFunc.addr,
		7,
		uintptr(s),
		uintptr(name),
		uintptr(namelen),
		uintptr(unsafe.Pointer(sendBuf)),
		uintptr(sendDataLen),
		uintptr(unsafe.Pointer(bytesSent)),
		uintptr(unsafe.Pointer(overlapped)),
		0,
		0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

//func ConnectEx(fd syscall.Handle, sa sockaddrInet, sendBuf *byte, sendDataLen uint32, bytesSent *uint32, overlapped *syscall.Overlapped) error {
//	err := LoadConnectEx()
//	if err != nil {
//		return errors.New("failed to find ConnectEx: " + err.Error())
//	}
//	ptr, n, err := sa.sockaddr()
//	if err != nil {
//		return err
//	}
//	//ol := syscall.Overlapped{}
//	//ol.Internal = overlapped.Internal
//	//ol.InternalHigh = overlapped.InternalHigh
//	//ol.Offset = overlapped.Offset
//	//ol.OffsetHigh = overlapped.OffsetHigh
//	//ol.HEvent = syscall.Handle(overlapped.HEvent)
//	return connectEx(fd, ptr, n, sendBuf, sendDataLen, bytesSent, overlapped)
//}

func ConnectEx(fd syscall.Handle, sa sockaddrInet, sendBuf *byte, sendDataLen uint32, bytesSent *uint32, overlapped *syscall.Overlapped) error {
	err := LoadConnectEx()
	if err != nil {
		return errors.New("failed to find ConnectEx: " + err.Error())
	}
	ptr, n, err := sa.sockaddr()
	if err != nil {
		return err
	}
	return connectEx(fd, ptr, n, sendBuf, sendDataLen, bytesSent, overlapped)
}
