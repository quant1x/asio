package v2

import (
	"fmt"
	"golang.org/x/sys/windows"
	"net"
	"strconv"
)

// 解析 URL（如 "example.com:80"）为 windows.Sockaddr
func resolveSockaddr(url string) (windows.Sockaddr, error) {
	// 分割主机和端口
	host, portStr, err := net.SplitHostPort(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %v", err)
	}

	// 解析端口
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %v", portStr)
	}

	// 解析主机名到 IP 地址
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %v", err)
	}

	// 优先选择 IPv4 地址（可根据需求调整）
	var ip net.IP
	for _, addr := range ips {
		if addr.To4() != nil {
			ip = addr
			break
		}
	}
	if ip == nil {
		ip = ips[0] // 如果没有 IPv4，选择第一个地址（可能是 IPv6）
	}

	// 根据 IP 类型生成 Sockaddr
	switch {
	case ip.To4() != nil:
		return &windows.SockaddrInet4{
			Port: port,
			Addr: [4]byte(ip.To4()),
		}, nil
	case ip.To16() != nil:
		return &windows.SockaddrInet6{
			Port: port,
			Addr: [16]byte(ip.To16()),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported IP type: %v", ip)
	}
}
