package main

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	WSAID_CONNECTEX = syscall.GUID{
		Data1: 0x25a207b9,
		Data2: 0xddf3,
		Data3: 0x4660,
		Data4: [8]byte{0x8e, 0xe9, 0x76, 0xe5, 0x8c, 0x74, 0x06, 0x3e},
	}
)

var (
	modws2_32                     = syscall.NewLazyDLL("kernel32.dll")
	procConnectEx                 = modws2_32.NewProc("ConnectEx")
	procCreateIoCompletionPort    = modws2_32.NewProc("CreateIoCompletionPort")
	procGetQueuedCompletionStatus = modws2_32.NewProc("GetQueuedCompletionStatus")
)

type Overlapped struct {
	Overlapped syscall.Overlapped
	Data       []byte
	OpType     int
}

const (
	OP_CONNECT = iota
	OP_READ
	OP_WRITE
)

func main() {
	// 初始化Winsock
	var wsaData syscall.WSAData
	syscall.WSAStartup(uint32(0x202), &wsaData)
	defer syscall.WSACleanup()

	// 创建IOCP
	hIOCP, err := syscall.CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 0)
	if hIOCP == 0 {
		panic("CreateIoCompletionPort failed")
	}

	// 创建工作者goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(uintptr(hIOCP), &wg)

	// 创建连接
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	// 关联到IOCP
	procCreateIoCompletionPort.Call(
		uintptr(fd),
		uintptr(hIOCP),
		0,
		0,
	)

	// 开始异步连接
	var overlapped Overlapped
	overlapped.OpType = OP_CONNECT
	addr := syscall.SockaddrInet4{
		Port: 8080,
		Addr: [4]byte{127, 0, 0, 1},
	}

	ret, _, _ := procConnectEx.Call(
		uintptr(fd),
		uintptr(unsafe.Pointer(&addr)),
		uintptr(unsafe.Sizeof(addr)),
		0,
		0,
		uintptr(unsafe.Pointer(&overlapped.Overlapped)),
	)

	if ret == 0 {
		err := syscall.GetLastError()
		if err != syscall.ERROR_IO_PENDING {
			panic(err)
		}
	}

	wg.Wait()
}

func worker(hIOCP uintptr, wg *sync.WaitGroup) {
	defer wg.Done()

	var (
		bytesTransferred uint32
		completionKey    uintptr
		overlapped       *Overlapped
	)

	for {
		ret, _, _ := procGetQueuedCompletionStatus.Call(
			hIOCP,
			uintptr(unsafe.Pointer(&bytesTransferred)),
			uintptr(unsafe.Pointer(&completionKey)),
			uintptr(unsafe.Pointer(&overlapped)),
			uintptr(syscall.INFINITE),
		)

		if ret == 0 {
			fmt.Println("IOCP error")
			continue
		}

		switch overlapped.OpType {
		case OP_CONNECT:
			fmt.Printf("Connected! Bytes transferred: %d\n", bytesTransferred)
			// 连接成功后投递读操作
			// 这里可以添加PostRecv等操作
		case OP_READ:
			fmt.Printf("Received %d bytes\n", bytesTransferred)
		case OP_WRITE:
			fmt.Printf("Sent %d bytes\n", bytesTransferred)
		}
	}
}
