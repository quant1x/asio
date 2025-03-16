package v2

import (
	"fmt"
	"golang.org/x/sys/windows"
	"testing"
)

func Test_resolveSockaddr(t *testing.T) {
	url := "baidu.com:443"
	addr, err := resolveSockaddr(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 打印结果（根据类型判断）
	switch addr := addr.(type) {
	case *windows.SockaddrInet4:
		fmt.Printf("IPv4 Address: %v, Port: %d\n", addr.Addr, addr.Port)
	case *windows.SockaddrInet6:
		fmt.Printf("IPv6 Address: %v, Port: %d\n", addr.Addr, addr.Port)
	}
}

func TestApacheBench(t *testing.T) {
	//var socklist []socket_t

}
