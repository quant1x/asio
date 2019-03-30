package asio

import (
	"fmt"
	"github.com/mymmsc/asio/os"
	"github.com/mymmsc/asio/reuseport"
	"syscall"
	"time"
)

type TcpSocket struct {
	fd int
}

var (
	// 无效的socket
	SOCKET_INVALID = -1
	// 超时30秒
	SOCKET_TIMEOUT int64 = 30
)

func Socket() (socket_t, error) {
	var (
		fd  int
		err error
	)
	syscall.ForkLock.RLock()
	if fd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP); err != nil {
		syscall.ForkLock.RUnlock()
		return SOCKET_INVALID, err
	}
	syscall.ForkLock.RUnlock()
	/*
	if err = syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return SOCKET_INVALID, err
	}*/
	return fd, nil
}

// 在non-blocking模式下,
// 如果返回值为-1, 且errno == EAGAIN或errno == EWOULDBLOCK表示no connections没有新连接请求
func Accept(fd socket_t) (socket_t, syscall.Sockaddr, error) {
	var (
		nfd socket_t
		sa  syscall.Sockaddr
		err error
	)
	for {
		nfd, sa, err = syscall.Accept(fd)
		//fmt.Printf("fd = %d, nfd = %d, error = %+v\n", fd, nfd, err)
		if nfd < 0 && (err == syscall.EAGAIN || err == syscall.EWOULDBLOCK) {
			return -1, nil, syscall.EAGAIN
		} else if nfd < 0 || err != nil {
			return -1, nil, err
		} else if nfd > 0 {
			break
		}
	}
	if err := syscall.SetNonblock(nfd, true); err != nil {
		return -1, nil, err
	}
	return nfd, sa, err
}

func Listen(addr string) (socket_t, error) {
	network, address := ParseAddr(addr)
	return reuseport.NewListener(network, address)
}

func Close(fd socket_t) error {
	if fd < 0 {
		return syscall.EBADF
	}
	return syscall.Close(fd)
}

func CloseEx(fd socket_t) error {
	if fd < 0 {
		return syscall.EBADF
	}
	syscall.Shutdown(fd, syscall.SHUT_WR)
	for {
		len := 4096
		buf := make([]byte, len)
		n, err := syscall.Read(fd, buf)
		if err != nil {
			n = 0
			// 如果返回EAGIN，阻塞当前协程直到有数据可读被唤醒
			if err == syscall.EAGAIN {
				continue
			}
		}
		if err != nil || n < 1 {
			break
		}
	}
	syscall.Shutdown(fd, syscall.SHUT_RD)
	return syscall.Close(fd)
}

// 在non-bloking模式下, 如果返回-1, 且errno = EINPROGRESS表示正在连接
func Connect(fd socket_t, addr string) error {
	sa, err := getAddr(addr)
	if err != nil {
		return err
	}
	return connect(fd, sa, time.Time{})
}

// this is close to the connect() function inside stdlib/net
func connect(fd socket_t, ra syscall.Sockaddr, deadline time.Time) error {
	switch err := syscall.Connect(fd, ra); err {
	case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
	case nil, syscall.EISCONN:
		if !deadline.IsZero() && deadline.Before(time.Now()) {
			return errTimeout
		}
		return nil
	default:
		return err
	}

	var err error
	var to syscall.Timeval
	var toptr *syscall.Timeval
	var pw syscall.FdSet
	FD_SET(uintptr(fd), &pw)
	for {
		// wait until the fd is ready to read or write.
		if !deadline.IsZero() {
			to = syscall.NsecToTimeval(deadline.Sub(time.Now()).Nanoseconds())
			toptr = &to
		}

		// wait until the fd is ready to write. we can't use:
		//   if err := fd.pd.WaitWrite(); err != nil {
		//   	 return err
		//   }
		// so we use select instead.
		if _, err = Select(fd+1, nil, &pw, nil, toptr); err != nil {
			fmt.Println(err)
			return err
		}

		var nerr int
		nerr, err = syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ERROR)
		if err != nil {
			return err
		}
		switch err = syscall.Errno(nerr); err {
		case syscall.EINPROGRESS, syscall.EALREADY, syscall.EINTR:
			continue
		case syscall.Errno(0), syscall.EISCONN:
			if !deadline.IsZero() && deadline.Before(time.Now()) {
				return errTimeout
			}
			return nil
		default:
			return err
		}
	}
}

// 在non-blocking模式下,
// 如果返回值为-1, 且errno == EAGAIN表示没有可接受的数据或很在接受尚未完成
func Recv(fd socket_t, data []byte) (int, error) {
	var (
		n      = 0
		err    error
		totlen = 0
	)
	count := len(data)
	for {
		//n, err = syscall.Read(fd, data)
		n, err = os.Recv(fd, data[totlen:])
		//fmt.Printf("fd = %d, n = %d, error = %+v\n", fd, n, err)
		//ngx_log_debug3(NGX_LOG_DEBUG_EVENT, c->log, 0, "recv: fd:%d %d of %d", c->fd, n, size);
		if n == 0 {
			//rev->ready = 0
			//rev->eof = 1
			return totlen, EOF
		} else if n > 0 {
			//data = append(data[:totlen], buf[0:n]...)
			totlen += n
			if totlen == count {
				return totlen, SUCCESS
			}
		} else {
			//err = ngx_socket_errno;
			if (err == syscall.EAGAIN || err == syscall.EWOULDBLOCK || err == syscall.EINTR) {
				//ngx_log_debug0(NGX_LOG_DEBUG_EVENT, c->log, err, "recv() not ready");
				err = EAGAIN
				if totlen > 0 {
					err = nil
				}
				/*if err == syscall.EINTR {
					continue
				}*/
				break
			} else {
				//n = ngx_connection_error(c, err, "recv() failed");
				err = ERROR
				break
			}
		}
	}

	//rev->ready = 0
	//if (n == NGX_ERROR) {
	//	rev->error = 1;
	//}

	return totlen, err
}

func Send(fd socket_t, data []byte) (int, error) {
	size := len(data)
	sent := 0
	ready := 1
	_error := 0
	n := 0
	var err error
	for {
		//n, err = syscall.Write(fd, data[sent:])
		n, err = os.Send(fd, data[sent:])
		//fmt.Printf("fd = %d, n = %d, error = %+v\n", fd, n, err)
		//ngx_log_debug3(NGX_LOG_DEBUG_EVENT, c->log, 0,	"send: fd:%d %d of %d", c->fd, n, size);
		if n > 0 {
			if n < size {
				ready = 0
			}
			sent += n
			if size == sent {
				err = SUCCESS
				break
			}
		} else if n == 0 {
			//wev->ready = 0;
			err = EOF
			break
		} else if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			err = EAGAIN
			break
		} else if err == syscall.EINTR {
			continue
		} else {
			_error = 1
			//(void) ngx_connection_error(c, err, "send() failed");
			err = ERROR
			break
		}
	}
	_ = ready
	_ = _error
	return sent, err
}

func SendTimeout(fd socket_t, data []byte, deadline time.Time) (int, error) {
	size := len(data)
	sent := 0
	ready := 1
	_error := 0
	n := 0
	var err error
	for {
		//n, err = syscall.Write(fd, data[sent:])
		n, err = os.Send(fd, data[sent:])
		//fmt.Printf("fd = %d, n = %d, error = %+v\n", fd, n, err)
		//ngx_log_debug3(NGX_LOG_DEBUG_EVENT, c->log, 0,	"send: fd:%d %d of %d", c->fd, n, size);
		if n > 0 {
			if n < size {
				ready = 0
			}
			sent += n
			if size == sent {
				err = SUCCESS
				break
			}
		} else if n == 0 {
			//wev->ready = 0;
			err = EOF
			break
		} else if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			err = EAGAIN
			break
		} else if err == syscall.EINTR {
			continue
		} else {
			_error = 1
			//(void) ngx_connection_error(c, err, "send() failed");
			err = ERROR
			break
		}
	}
	_ = ready
	_ = _error
	return sent, err
}
