package asio

import (
	"net"
	"strconv"
	"strings"
	"syscall"
)

func ParseAddr(addr string) (network, address string) {
	network = "tcp"
	address = addr
	if strings.Contains(address, "://") {
		network = strings.Split(address, "://")[0]
		address = strings.Split(address, "://")[1]
	}
	return
}

func getAddr(addr string) (syscall.Sockaddr, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	bits := strings.Split(tcpAddr.IP.String(), ".")

	b0, err := strconv.Atoi(bits[0])
	b1, err := strconv.Atoi(bits[1])
	b2, err := strconv.Atoi(bits[2])
	b3, err := strconv.Atoi(bits[3])

	sa := &syscall.SockaddrInet4{
		Port: tcpAddr.Port,
		Addr: [4]byte{byte(b0), byte(b1), byte(b2), byte(b3)},
	}
	return sa, nil
}
