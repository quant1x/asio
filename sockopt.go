package asio

import (
	"syscall"
	"time"
)

func Setsockopt(fd socket_t) error {
	var err error
	// 设置地址复用
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		//syscall.Close(fd)
		return err
	}

	// 设置端口复用
	if SO_REUSEPORT > 0 {
		if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_REUSEPORT, 1); err != nil {
			//syscall.Close(fd)
			return err
		}
	}
	// 设置超时
	tv := syscall.NsecToTimeval(SOCKET_TIMEOUT * time.Second.Nanoseconds())
	// 读 超时
	//SO_SNDTIMEO
	if err = syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); err != nil {
		//syscall.Close(fd)
		return err
	}
	// 写 超时
	if err = syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_SNDTIMEO, &tv); err != nil {
		//syscall.Close(fd)
		return err
	}
	/*
	// no sigpipe
	opt := 1
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_NOSIGPIPE, opt); err != nil {
		syscall.AE_CLOSE(fd)
		return nil
	}*/
	if err := syscall.SetNonblock(fd, true); err != nil {
		return err
	}
	bufSize := 64 * 1024
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, bufSize); err != nil {
		return err
	}
	if err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, bufSize); err != nil {
		return err
	}
	/*linger := syscall.Linger{
		Onoff:1,
		Linger:10,
	}
	syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)*/
	return nil
}

func SetsockoptLinger(fd socket_t, ttl int32) {
	linger := syscall.Linger{
		Onoff:  1,
		Linger: ttl,
	}
	syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
}
