package v2

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func Test_socket(t *testing.T) {
	s := socket()
	fmt.Println(s)
}

func TestIoURing(t *testing.T) {
	ring := create_engine()
	s := socket()
	socket_opt(s)
	err := engine_bind(ring, s)
	if err != nil {
		fmt.Println(err)
	}
	sa := windows.SockaddrInet4{
		Port: 8080,
		Addr: [4]byte{127, 0, 0, 1},
	}
	ov := Overlapped{
		Overlapped: windows.Overlapped{},
		OP:         OpConnect,
		socket:     s,
	}
	err = connect(s, &sa, &ov)
	if err != nil && !errors.Is(err, syscall.ERROR_IO_PENDING) {
		panic(err)
	}

	// 处理完成端口事件
	go func() {
		var bytesTransferred uint32
		var completionKey uintptr
		var rawOverlapped *windows.Overlapped

		for {
			err := windows.GetQueuedCompletionStatus(ring, &bytesTransferred, &completionKey, &rawOverlapped, windows.INFINITE)
			if err != nil {
				if errors.Is(err, windows.WAIT_TIMEOUT) {
					continue
				}
				log.Println("GetQueuedCompletionStatus error:", err)
				return
			}
			overlapped := (*Overlapped)(unsafe.Pointer(rawOverlapped))
			if overlapped == nil {
				log.Println("Received shutdown signal")
				return
			}
			switch overlapped.OP {
			case OpConnect:
				log.Println("Received connect")
				{
					// 示例：发送数据
					sendBuf := []byte("GET / HTTP/1.0\r\n\r\n")
					var sendOverlapped = Overlapped{
						Overlapped: windows.Overlapped{},
						OP:         OpWrite,
						socket:     overlapped.socket,
						//buf:        &sendBuf[0],
					}
					var sendBufs []windows.WSABuf
					sendBufs = append(sendBufs, windows.WSABuf{
						Len: uint32(len(sendBuf)),
						Buf: &sendBuf[0],
					})

					var flags uint32
					err = windows.WSASend(overlapped.socket, &sendBufs[0], 1, nil, flags, &sendOverlapped.Overlapped, nil)
					if err != nil && err != syscall.ERROR_IO_PENDING {
						log.Fatal("WSASend failed:", err)
					}
				}
			case OpClosed:
				log.Println("Received close")
			case OpRead:
				log.Println("Received read", string(overlapped.buf))
				windows.Closesocket(overlapped.socket)
			case OpWrite:
				log.Println("Received write")
				{
					var recvOverlapped = Overlapped{
						Overlapped: windows.Overlapped{},
						OP:         OpRead,
						socket:     overlapped.socket,
						buf:        make([]byte, 1024),
					}
					var recvBufs []windows.WSABuf
					recvBufs = append(recvBufs, windows.WSABuf{
						Len: uint32(len(recvOverlapped.buf)),
						Buf: &recvOverlapped.buf[0],
					})

					var flags uint32
					err = windows.WSARecv(overlapped.socket, &recvBufs[0], 1, nil, &flags, &recvOverlapped.Overlapped, nil)
					if err != nil && err != syscall.ERROR_IO_PENDING {
						log.Fatal("WSARecv failed:", err)
					}
				}
			default:
				log.Println("Received unknown op")
			}
			runtime.KeepAlive(ov)
			// 处理连接完成
			log.Printf("Operation completed, bytes transferred: %d", bytesTransferred)

			// 这里可以添加发送/接收数据的逻辑
		}
	}()
	select {}
}
