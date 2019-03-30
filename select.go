package asio

import (
	"fmt"
	"syscall"
	"time"
)

func FD_SET(fd uintptr, p *syscall.FdSet) {
	n, k := fd/32, fd%32
	p.Bits[n] |= (1 << uint32(k))
}

func CheckoutTimeout(fd socket_t, deadline time.Time) (error) {
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
		case syscall.ETIMEDOUT:
			return errTimeout
		case syscall.EAGAIN:
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
