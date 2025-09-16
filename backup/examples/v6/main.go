package main

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetQueuedCompletionStatusEx = modkernel32.NewProc("GetQueuedCompletionStatusEx")
)

type overlapped struct {
	internal     uintptr
	internalhigh uintptr
	anon0        [8]byte
	hevent       *byte
}

type overlappedEntry struct {
	key      uintptr
	ov       *overlapped
	internal uintptr
	qty      uint32
}

type OverlappedEntry struct {
	lpCompletionKey            uintptr
	lpOverlapped               *overlapped
	Internal                   uintptr
	dwNumberOfBytesTransferred uint32
}

func main() {
	// 创建完成端口
	hPort, err := windows.CreateIoCompletionPort(windows.InvalidHandle, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	defer windows.CloseHandle(hPort)

	// 启动事件循环
	go func() {
		entries := make([]OverlappedEntry, 10)
		var numEntries uint32

		for {
			success, err := GetQueuedCompletionStatusEx(
				hPort,
				entries,
				&numEntries,
				windows.INFINITE,
				false,
			)

			if !success {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			for i := uint32(0); i < numEntries; i++ {
				fmt.Printf("Completion Key: %d, Bytes: %d\n",
					entries[i].lpCompletionKey,
					entries[i].dwNumberOfBytesTransferred,
				)
			}
		}
	}()

	// 模拟 PostQueuedCompletionStatus
	time.Sleep(time.Second)
	windows.PostQueuedCompletionStatus(hPort, 100, 0x1234, nil)
	select {}
}

func GetQueuedCompletionStatusEx(
	hCompletionPort windows.Handle,
	entries []OverlappedEntry,
	numEntries *uint32,
	timeout uint32,
	alertable bool,
) (bool, error) {

	alertableFlag := uintptr(0)
	if alertable {
		alertableFlag = 1
	}

	ret, _, err := procGetQueuedCompletionStatusEx.Call(
		uintptr(hCompletionPort),
		uintptr(unsafe.Pointer(&entries[0])),
		uintptr(len(entries)),
		uintptr(unsafe.Pointer(numEntries)),
		uintptr(timeout),
		alertableFlag,
	)

	return ret != 0, err
}
