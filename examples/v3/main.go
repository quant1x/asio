package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
	_ "unsafe"
)

//go:linkname CreateIoCompletionPort syscall.createIoCompletionPort
func CreateIoCompletionPort(fileHandle syscall.Handle, cpHandle syscall.Handle, key uintptr, threadCnt uint32) (handle syscall.Handle, err error)

//go:linkname GetQueuedCompletionStatus syscall.getQueuedCompletionStatus
func GetQueuedCompletionStatus(cpHandle syscall.Handle, qty *uint32, key *uintptr, overlapped **syscall.Overlapped, timeout uint32) (err error)

//go:linkname PostQueuedCompletionStatus syscall.postQueuedCompletionStatus
func PostQueuedCompletionStatus(cphandle syscall.Handle, qty uint32, key uintptr, overlapped *syscall.Overlapped) (err error)

// //go:linkname errnoErr syscall.errnoErr
func errnoErr(e syscall.Errno) error {
	return syscall.Errno(e)
}

var (
	modws2_32 = syscall.NewLazyDLL("Ws2_32.dll")

	procWSACreateEvent           = modws2_32.NewProc("WSACreateEvent")
	procWSACloseEvent            = modws2_32.NewProc("WSACloseEvent")
	procWSAWaitForMultipleEvents = modws2_32.NewProc("WSAWaitForMultipleEvents")
	procWSAResetEvent            = modws2_32.NewProc("WSAResetEvent")
	procWSAGetOverlappedResult   = modws2_32.NewProc("WSAGetOverlappedResult")
)

func WSACreateEvent() (handle syscall.Handle, err error) {
	r1, _, e1 := syscall.SyscallN(procWSACreateEvent.Addr())
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return syscall.Handle(r1), err
}

func WSACloseEvent(handle syscall.Handle) (err error) {
	r1, _, e1 := syscall.SyscallN(procWSACloseEvent.Addr(), uintptr(handle))
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return err
}

func WSAResetEvent(handle syscall.Handle) (err error) {
	r1, _, e1 := syscall.SyscallN(procWSAResetEvent.Addr(), uintptr(handle))
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return
}

func WSAWaitForMultipleEvents(cEvents uint32, lpEvent *syscall.Handle, fWaitAll bool, dwTimeout uint32, fAlertable bool) (uint32, error) {
	var WaitAll, Alertable uint32
	if fWaitAll {
		WaitAll = 1
	}
	if fAlertable {
		Alertable = 1
	}
	r1, _, e1 := syscall.SyscallN(procWSAWaitForMultipleEvents.Addr(), uintptr(cEvents), uintptr(unsafe.Pointer(lpEvent)), uintptr(WaitAll), uintptr(dwTimeout), uintptr(Alertable))
	if r1 == syscall.WAIT_FAILED {
		return 0, errnoErr(e1)
	}
	return uint32(r1), nil
}

func WSAGetOverlappedResult(socket syscall.Handle, overlapped *syscall.Overlapped, transferBytes *uint32, bWait bool, flag *uint32) (err error) {
	var wait uint32
	if bWait {
		wait = 1
	}
	r1, _, e1 := syscall.SyscallN(procWSAGetOverlappedResult.Addr(), uintptr(socket), uintptr(unsafe.Pointer(overlapped)),
		uintptr(unsafe.Pointer(transferBytes)), uintptr(wait), uintptr(unsafe.Pointer(flag)))
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return
}

func SetNonBlock(fd syscall.Handle) error {
	flag := uint32(1)
	size := uint32(unsafe.Sizeof(flag))
	ret := uint32(0)
	ol := syscall.Overlapped{}
	err := syscall.WSAIoctl(fd, 0x8004667e, (*byte)(unsafe.Pointer(&flag)), size, nil, 0, &ret, &ol, 0)
	if err != nil {
		return err
	}
	return nil
}

type IOData struct {
	Overlapped syscall.Overlapped
	WsaBuf     syscall.WSABuf
	NBytes     uint32
	isRead     bool
	cliSock    syscall.Handle
}

func closeIO(data *IOData) {
	if data.Overlapped.HEvent != syscall.Handle(0) {
		WSACloseEvent(data.Overlapped.HEvent)
		data.Overlapped.HEvent = syscall.Handle(0)
	}
	syscall.Closesocket(data.cliSock)
}

func main() {
	listenFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return
	}
	defer func() {
		syscall.Closesocket(listenFd)
		syscall.WSACleanup()
	}()
	v4 := &syscall.SockaddrInet4{
		Port: 6000,
		Addr: [4]byte{},
	}
	err = syscall.Bind(listenFd, v4)
	if err != nil {
		return
	}
	err = syscall.Listen(listenFd, 0)
	if err != nil {
		return
	}

	hIOCP, err := CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 0)
	if err != nil {
		return
	}
	count := runtime.NumCPU()
	for i := 0; i < count; i++ {
		go workThread(hIOCP)
	}

	defer func() {
		for i := 0; i < count; i++ {
			PostQueuedCompletionStatus(hIOCP, 0, 0, nil)
		}
	}()

	for {
		acceptFd, er := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
		if er != nil {
			break
		}
		b := make([]byte, 1024)
		recvD := uint32(0)
		data := &IOData{
			Overlapped: syscall.Overlapped{},
			WsaBuf: syscall.WSABuf{
				Len: 1024,
				Buf: &b[0],
			},
			NBytes:  1024,
			isRead:  true,
			cliSock: acceptFd,
		}
		data.Overlapped.HEvent, er = WSACreateEvent()
		if er != nil {
			fmt.Printf("WSACreateEvent failed:%s", er)
			closeIO(data)
			break
		}

		size := uint32(unsafe.Sizeof(&syscall.SockaddrInet4{}) + 16)
		er = syscall.AcceptEx(listenFd, acceptFd, data.WsaBuf.Buf, data.WsaBuf.Len-size*2, size, size, &recvD, &data.Overlapped)
		if er != nil && !errors.Is(er, syscall.ERROR_IO_PENDING) {
			er = os.NewSyscallError("AcceptEx", er)
			fmt.Printf("AcceptEx Error:%s", er)
			closeIO(data)
			break
		}

		_, er = WSAWaitForMultipleEvents(1, &data.Overlapped.HEvent, true, 100, false)
		if er != nil {
			fmt.Printf("WSAWaitForMultipleEvents Error:%s", er)
			closeIO(data)
			break
		}
		WSAResetEvent(data.Overlapped.HEvent)
		flag := uint32(0)
		er = WSAGetOverlappedResult(acceptFd, &data.Overlapped, &data.NBytes, true, &flag)
		if er != nil {
			fmt.Printf("WSAGetOverlappedResult Error:%s", er)
			closeIO(data)
			break
		}
		if data.NBytes == 0 {
			closeIO(data)
			continue
		}
		fmt.Printf("client %d connected\n", acceptFd)
		_, err = CreateIoCompletionPort(acceptFd, hIOCP, 0, 0)
		if err != nil {
			fmt.Printf("CreateIoCompletionPort Error:%s", er)
			closeIO(data)
			break
		}
		postWrite(data)
	}
}

func postWrite(data *IOData) (err error) {
	data.isRead = false
	// 这里输出一下data指针，让运行时不把data给GC掉，否则就会出问题
	fmt.Printf("%p cli:%d send %s\n", data, data.cliSock, unsafe.String(data.WsaBuf.Buf, data.NBytes))
	err = syscall.WSASend(data.cliSock, &data.WsaBuf, 1, &data.NBytes, 0, &data.Overlapped, nil)
	if err != nil && !errors.Is(err, syscall.ERROR_IO_PENDING) {
		fmt.Printf("cli:%d send failed: %s\n", data.cliSock, err)
		closeIO(data)
		return err
	}
	return
}

func postRead(data *IOData) (err error) {
	data.NBytes = data.WsaBuf.Len
	data.isRead = true
	flag := uint32(0)
	err = syscall.WSARecv(data.cliSock, &data.WsaBuf, 1, &data.NBytes, &flag, &data.Overlapped, nil)
	if err != nil && !errors.Is(err, syscall.ERROR_IO_PENDING) {
		fmt.Printf("cli:%d receive failed: %s\n", data.cliSock, err)
		closeIO(data)
		return err
	}
	return
}

func workThread(hIOCP syscall.Handle) {
	var pOverlapped *syscall.Overlapped
	var ioSize uint32
	var key uintptr
	for {
		err := GetQueuedCompletionStatus(hIOCP, &ioSize, &key, &pOverlapped, syscall.INFINITE)
		if err != nil {
			fmt.Printf("GetQueuedCompletionStatus failed: %s\n", err)
			return
		}
		ioData := (*IOData)(unsafe.Pointer(pOverlapped))
		if ioSize == 0 {
			closeIO(ioData)
			break
		}
		ioData.NBytes = ioSize
		if ioData.isRead {
			postWrite(ioData)
		} else {
			postRead(ioData)
		}
	}
}
