package v2

import (
	"log"

	"golang.org/x/sys/windows"
)

// 跨平台的套接字类型重定向
type rawSocket = windows.Handle

// 跨平台的io_uring
type rawRing = windows.Handle

type Overlapped struct {
	Overlapped windows.Overlapped
	OP         OpType
	socket     rawSocket
	buf        []byte
}

func init() {
	// 初始化Winsock
	var wsaData windows.WSAData
	if err := windows.WSAStartup(2<<16|2, &wsaData); err != nil {
		log.Fatal("WSAStartup failed:", err)
	}
}

// 创建socket
func socket() rawSocket {
	// 创建socket
	s, err := windows.WSASocket(windows.AF_INET, windows.SOCK_STREAM, windows.IPPROTO_TCP, nil, 0, windows.WSA_FLAG_OVERLAPPED)
	if err != nil {
		log.Fatal("WSASocket failed:", err)
	}
	return s
}

func connect(s rawSocket, sa windows.Sockaddr, ov *Overlapped) error {
	err := windows.ConnectEx(s, sa, nil, 0, nil, &ov.Overlapped)
	return err
}

func create_engine() rawRing {
	// 创建IOCP
	iocp, err := windows.CreateIoCompletionPort(windows.InvalidHandle, 0, 0, 0)
	if err != nil {
		log.Fatal("CreateIoCompletionPort failed:", err)
	}
	return iocp
}

func engine_close(fd rawRing) {
	windows.CloseHandle(fd)
}

func engine_bind(h rawRing, s rawSocket) error {
	// 绑定本地地址（ConnectEx要求）
	localAddr := windows.SockaddrInet4{Port: 0}
	if err := windows.Bind(s, &localAddr); err != nil {
		return err
	}
	// 将socket关联到IOCP
	_iocp, err := windows.CreateIoCompletionPort(s, h, uintptr(s), 0)
	if err != nil {
		return err
	}
	_ = _iocp
	return nil
}
