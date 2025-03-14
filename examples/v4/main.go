package main

/*
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>

#pragma comment(lib, "ws2_32.lib")

static SOCKET createSocket() {
    return socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
}

static int asyncConnect(SOCKET sock, const char* ip, int port, HANDLE hEvent) {
    struct sockaddr_in addr;
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    inet_pton(AF_INET, ip, &addr.sin_addr);

    // 设置非阻塞模式
    u_long mode = 1;
    ioctlsocket(sock, FIONBIO, &mode);

    // 绑定事件
    WSAEventSelect(sock, hEvent, FD_CONNECT | FD_CLOSE);

    return connect(sock, (struct sockaddr*)&addr, sizeof(addr));
}
*/
import "C"
import (
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	// 初始化 Winsock
	var wsaData C.WSADATA
	if C.WSAStartup(C.MAKEWORD(2, 2), &wsaData) != 0 {
		panic("WSAStartup failed")
	}
	defer C.WSACleanup()

	// 创建事件对象
	hEvent := C.WSACreateEvent()
	defer C.WSACloseEvent(hEvent)

	// 创建 Socket
	sock := C.createSocket()
	defer C.closesocket(sock)

	// 异步连接
	ip := C.CString("127.0.0.1")
	defer C.free(unsafe.Pointer(ip))
	if C.asyncConnect(sock, ip, 8080, hEvent) != 0 {
		if errno := C.WSAGetLastError(); errno != C.WSAEWOULDBLOCK {
			panic(fmt.Sprintf("connect error: %d", errno))
		}
	}

	// 启动事件循环
	resultCh := make(chan error)
	go func() {
		defer close(resultCh)

		// 等待事件触发
		ret := C.WSAWaitForSingleEvent(hEvent, C.INFINITE, C.FALSE)
		if ret == C.WSA_WAIT_FAILED {
			resultCh <- fmt.Errorf("WSAWaitForSingleEvent failed")
			return
		}

		// 获取网络事件
		var networkEvents C.WSANETWORKEVENTS
		if C.WSAEnumNetworkEvents(sock, hEvent, &networkEvents) != 0 {
			resultCh <- fmt.Errorf("WSAEnumNetworkEvents failed")
			return
		}

		// 处理连接结果
		if (networkEvents.lNetworkEvents & C.FD_CONNECT) != 0 {
			if networkEvents.iErrorCode[C.FD_CONNECT_BIT] != 0 {
				resultCh <- fmt.Errorf("connect failed: %d", networkEvents.iErrorCode[C.FD_CONNECT_BIT])
			} else {
				resultCh <- nil // 连接成功
			}
		}
	}()

	// 等待结果
	select {
	case err := <-resultCh:
		if err != nil {
			fmt.Println("连接失败:", err)
		} else {
			fmt.Println("连接成功!")
			// 转换为 Go 的 net.Conn 以便后续操作
			conn := &winSocketConn{
				fd: syscall.Handle(sock),
				la: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0},
				ra: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080},
			}
			_ = conn // 使用 conn 进行读写操作
		}
	}
}

// 实现 net.Conn 接口的包装器
type winSocketConn struct {
	fd     syscall.Handle
	la, ra net.Addr
}

func (c *winSocketConn) Read(b []byte) (n int, err error) {
	return syscall.Read(c.fd, b)
}

func (c *winSocketConn) Write(b []byte) (n int, err error) {
	return syscall.Write(c.fd, b)
}

func (c *winSocketConn) Close() error {
	return syscall.Close(c.fd)
}

func (c *winSocketConn) LocalAddr() net.Addr                { return c.la }
func (c *winSocketConn) RemoteAddr() net.Addr               { return c.ra }
func (c *winSocketConn) SetDeadline(t time.Time) error      { return nil }
func (c *winSocketConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *winSocketConn) SetWriteDeadline(t time.Time) error { return nil }
