// +build darwin netbsd freebsd openbsd dragonfly

package asio

import (
	"syscall"
	"time"
)

//type kqueue_event = syscall.Kevent_t

type Poll struct {
	index  int
	fd     int
	events []syscall.Kevent_t
}

func OpenPoll(index int) (*Poll, error) {
	poll := &Poll{index: index}
	fd, err := syscall.Kqueue()
	if err == nil {
		poll.fd = fd
	}

	return poll, err
}

// Close ...
func (p *Poll) Close() error {
	return syscall.Close(p.fd)
}

func (poll *Poll) Modify(ev *Event, mask int) (error) {
	var (
		op  uint16
		n   int
		err error
	)
	if (mask & AE_ADD) > 0 {
		op = syscall.EV_ADD
	} else if (mask & AE_DEL) > 0 {
		op = syscall.EV_DELETE
	}
	if (mask & AE_READABLE) > 0 {
		n, err = syscall.Kevent(poll.fd,
			[]syscall.Kevent_t{{Ident: uint64(ev.Fd),
				Flags: op, Filter: syscall.EVFILT_READ}},
			nil, nil)
		if op == syscall.EV_ADD {
			ev.mask |= AE_READABLE
		} else {
			ev.mask &= ^ AE_READABLE
		}
	}
	if (mask & AE_WRITABLE) > 0 {
		n, err = syscall.Kevent(poll.fd,
			[]syscall.Kevent_t{{Ident: uint64(ev.Fd),
				Flags: op, Filter: syscall.EVFILT_WRITE}},
			nil, nil)
		if op == syscall.EV_ADD {
			ev.mask |= AE_WRITABLE
		} else {
			ev.mask &= ^ AE_WRITABLE
		}
	}
	_ = n
	return err
}

func (p *Poll) Wait(changes map[int]*Event, events []*Event, timeout time.Duration) (n int, err error) {
	if timeout < 0 {
		timeout = 0
	}
	if len(p.events) < len(events) {
		p.events = make([]syscall.Kevent_t, len(events))
		//p.events = make([]syscall.Kevent_t, 128)
	}
	pos := 0
	//ts := syscall.NsecToTimespec(int64(timeout))
	for {
		n, err := syscall.Kevent(p.fd, nil, p.events, nil)
		if err != nil {
			if temporaryErr(err) {
				continue
			}
			return n, err
		}

		for i := 0; i < n; i++ {
			event := p.events[i]
			fd := socket_t(event.Ident)
			//fmt.Printf("event %+v \n", event)
			if fd < 0 {
				continue
			}
			ev, ok := changes[fd]
			if !ok {
				continue
			}
			mask := 0
			if event.Flags & syscall.EV_ERROR == syscall.EV_ERROR || event.Flags & syscall.EV_EOF == syscall.EV_EOF {
				//fmt.Printf("[%d: EV_ERROR]event = %+v\n", p.index, event)
				err := syscall.Errno(event.Data);
				if err == syscall.ENOENT { // resubmit changes on ENOENT
					p.Modify(ev, ev.mask)
				} else if err == syscall.EBADF {
					if FD_VALID(fd) {
						if err == syscall.ENOENT { // on EBADF, we re-check the fd
							p.Modify(ev, ev.mask)
						} else {
							mask |= AE_ERROR
						}
					}
				} else {
					mask |= AE_ERROR
				}
			}

			if event.Filter == syscall.EVFILT_READ {
				mask |= AE_READABLE
			}
			if event.Filter == syscall.EVFILT_WRITE {
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
