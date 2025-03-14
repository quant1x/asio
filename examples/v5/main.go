package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	modWs2_32        = syscall.NewLazyDLL("ws2_32.dll")
	proConnectEx     = modWs2_32.NewProc("ConnectEx")
	modKernel32      = syscall.NewLazyDLL("kernel32.dll")
	connectExFuncPtr uintptr
)

func main() {
	// 创建TCP套接字
	socket, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		panic(err)
	}
	defer syscall.Close(socket)
	//// 获取ConnectEx函数指针
	//err = getConnectExFunc(syscall.Handle(socket))
	//if err != nil {
	//	panic(err)
	//}
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
	// 准备连接地址
	//address := syscall.SockaddrInet4{
	//	Port: int(htons(8080)),
	//	Addr: [4]byte{127, 0, 0, 1},
	//}
	//sa, saLen, _ := getRawSockaddr(&address)
	// 目标地址
	sa := sockaddrInet{
		Family: syscall.AF_INET,
		Port:   uint16(8080),
		Addr:   [4]byte{127, 0, 0, 1},
	}
	//sa.Port = htons(80)
	//*(*uint16)(unsafe.Pointer(&sa.Port)) = htons(sa.Port)

	ol := &syscall.Overlapped{}
	err = ConnectEx(socket, sa, nil, 0, nil, ol)
	if err != nil && !errors.Is(err, syscall.ERROR_IO_PENDING) {
		panic(err)
	}
	//overlapped := &windows.Overlapped{}
	//_, _, errno := syscall.SyscallN(
	//	connectExFuncPtr, //proConnectEx.Addr(),
	//	uintptr(socket),
	//	uintptr(unsafe.Pointer(&sa)),
	//	uintptr(unsafe.Sizeof(sa)),
	//	0, 0, uintptr(unsafe.Pointer(overlapped)),
	//)
	////syscall.ConnectEx()
	//if errno != 0 {
	//	if errno != syscall.ERROR_IO_PENDING {
	//		panic(errno)
	//	}
	//}

	// 等待连接完成（实际应在事件循环处理）
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

type SockaddrInet4 struct {
	Port int
	Addr [4]byte
	raw  syscall.RawSockaddrInet4
}

func (sa *SockaddrInet4) sockaddr() (unsafe.Pointer, int32, error) {
	if sa.Port < 0 || sa.Port > 0xFFFF {
		return nil, 0, syscall.EINVAL
	}
	sa.raw.Family = syscall.AF_INET
	p := (*[2]byte)(unsafe.Pointer(&sa.raw.Port))
	p[0] = byte(sa.Port >> 8)
	p[1] = byte(sa.Port)
	sa.raw.Addr = sa.Addr
	return unsafe.Pointer(&sa.raw), int32(unsafe.Sizeof(sa.raw)), nil
}

var (
	SIO_GET_EXTENSION_FUNCTION_POINTER = uint32(0xc8000006)
	WSAID_CONNECTEX                    = windows.GUID{0x25a207b9, 0xddf3, 0x4660, [8]byte{0x8e, 0xe9, 0x76, 0xe5, 0x8c, 0x74, 0x06, 0x3e}}
)

func getConnectExFunc(s syscall.Handle) error {
	var bytesReturned uint32
	err := windows.WSAIoctl(
		windows.Handle(s),
		SIO_GET_EXTENSION_FUNCTION_POINTER,
		(*byte)(unsafe.Pointer(&WSAID_CONNECTEX)),
		uint32(unsafe.Sizeof(WSAID_CONNECTEX)),
		(*byte)(unsafe.Pointer(&connectExFuncPtr)),
		uint32(unsafe.Sizeof(connectExFuncPtr)),
		&bytesReturned,
		nil,
		0,
	)
	return err
}

// 手动实现htons
func htons(n uint16) uint16 {
	return (n << 8) | (n >> 8)
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
			SIO_GET_EXTENSION_FUNCTION_POINTER,
			(*byte)(unsafe.Pointer(&WSAID_CONNECTEX)),
			uint32(unsafe.Sizeof(WSAID_CONNECTEX)),
			(*byte)(unsafe.Pointer(&connectExFunc.addr)),
			uint32(unsafe.Sizeof(connectExFunc.addr)),
			&n, nil, 0)
	})
	return connectExFunc.err
}

func connectEx(s syscall.Handle, name unsafe.Pointer, namelen int32, sendBuf *byte, sendDataLen uint32, bytesSent *uint32, overlapped *syscall.Overlapped) (err error) {
	r1, _, e1 := syscall.Syscall9(connectExFunc.addr, 7, uintptr(s), uintptr(name), uintptr(namelen), uintptr(unsafe.Pointer(sendBuf)), uintptr(sendDataLen), uintptr(unsafe.Pointer(bytesSent)), uintptr(unsafe.Pointer(overlapped)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

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
