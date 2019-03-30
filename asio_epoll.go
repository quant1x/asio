// +build linux

package asio

import (
	"fmt"
	"syscall"
	"time"
)

// https://www.jianshu.com/p/7835726dc78b
// https://blog.csdn.net/linuxheik/article/details/73294658

// Poll ...
type Poll struct {
	index  int
	fd     int // epoll fd
	wfd    int // wake fd
	events []syscall.EpollEvent
}

func OpenPoll(index int) (*Poll, error) {
	poll := &Poll{index: index}
	//p, err := syscall.EpollCreate1(0)
	p, err := syscall.EpollCreate(1024)
	if err != nil {
		panic(err)
	}
	poll.fd = p
	/*r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(p)
		panic(err)
	}
	poll.wfd = int(r0)
	poll.AddRead(poll.wfd)*/
	return poll, nil
}

// Close ...
func (p *Poll) Close() error {
	/*if err := syscall.Close(p.wfd); err != nil {
		return err
	}*/
	return syscall.Close(p.fd)
}

// Trigger ...
func (p *Poll) Trigger(note interface{}) error {
	_, err := syscall.Write(p.wfd, []byte{0, 0, 0, 0, 0, 0, 0, 1})
	return err
}

func (poll *Poll) Modify(ev *Event, mask int) (error) {
	var (
		op  int
		n   int
		err error
	)

	if (mask & AE_ADD) > 0 {
		op = syscall.EPOLL_CTL_ADD
		if ev.mask > 0 {
			op = syscall.EPOLL_CTL_MOD
		}
	} else if (mask & AE_DEL) > 0 {
		op = syscall.EPOLL_CTL_DEL
	}
	if (mask & AE_READABLE) > 0 {
		err = syscall.EpollCtl(poll.fd, op, ev.Fd,
			&syscall.EpollEvent{Fd: int32(ev.Fd),
				Events: syscall.EPOLLIN,
			},
		)
		if op == syscall.EPOLL_CTL_ADD || op == syscall.EPOLL_CTL_MOD {
			ev.mask |= AE_READABLE
		} else {
			ev.mask &= ^ AE_READABLE
		}
	}
	if (mask & AE_WRITABLE) > 0 {
		err = syscall.EpollCtl(poll.fd, op, ev.Fd,
			&syscall.EpollEvent{Fd: int32(ev.Fd),
				Events: syscall.EPOLLOUT,
			},
		)
		if op == syscall.EPOLL_CTL_ADD || op == syscall.EPOLL_CTL_MOD {
			ev.mask |= AE_WRITABLE
		} else {
			ev.mask &= ^ AE_WRITABLE
		}
	}
	_ = n
	return err
}

// Wait ...
func (p *Poll) Wait(changes map[int]*Event, events []*Event, timeout time.Duration) (n int, err error) {
	if timeout < 0 {
		timeout = 0
	}
	if len(p.events) < len(events) {
		p.events = make([]syscall.EpollEvent, len(events))
	}
	pos := 0
	for {
		n, err := syscall.EpollWait(p.fd, p.events, int(timeout))
		//fmt.Printf("pn = %d, event=%+v\n", n, err)
		if err != nil {
			if temporaryErr(err) {
				continue
			}
			return n, err
		}
		/*if err != nil && err != syscall.EINTR {
			return n, err
		}*/
		for i := 0; i < n; i++ {
			event := p.events[i]
			fd := int(event.Fd)
			//fmt.Printf("event %+v \n", event)
			if fd < 0 {
				continue
			}
			ev, ok := changes[fd]
			if !ok {
				continue
			}
			_ = ev
			mask := 0
			// 非关注的事件, 全部当作异常处理
			// TODO: EPOLLWAKEUP 需要处理
			if event.Events & syscall.EPOLLIN == 0 && event.Events & syscall.EPOLLOUT == 0 && event.Events & syscall.EPOLLHUP == 0 {
				fmt.Printf("[%d: EV_ERROR]event = %+v\n", p.index, event)
				mask |= AE_ERROR
			}
			if event.Events == syscall.EPOLLIN || event.Events == syscall.EPOLLHUP {
				mask |= AE_READABLE
			}
			if event.Events == syscall.EPOLLOUT {
				mask |= AE_WRITABLE
			}
			e := &Event{
				Fd:   fd,
				mask: mask,
			}
			events[pos] = e
			pos ++
		}
		break
	}
	return pos, nil
}
