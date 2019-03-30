package asio

import (
	"fmt"
	"syscall"
	"time"
)

const (
	AE_NONE     = 0x0000 /* No events registered. */
	AE_ERROR    = 0x0008
	AE_ADD      = 0x1000 // 添加事件
	AE_DEL      = 0x2000 // 删除事件
	AE_READABLE = 0x0001 /* Fire when descriptor is readable. */
	AE_WRITABLE = 0x0002 /* Fire when descriptor is writable. */
	AE_BARRIER  = 0x0004 /* With WRITABLE, never fire the event if the
                           READABLE event already fired in the same event
                           loop iteration. Useful when you want to persist
                           things to disk before sending replies, and want
                           to do that in a group fashion. */
)

type Status int

const (
	INVALID      = socket_t(-1)
	//SUCCESS   Status = iota
	__errno_base = 90000
	ERROR        = syscall.Errno(__errno_base + 1)
	EAGAIN       = syscall.Errno(__errno_base + 2)
	EBUSY        = syscall.Errno(__errno_base + 3)
	EDONE        = syscall.Errno(__errno_base + 4)
	EDECLINED    = syscall.Errno(__errno_base + 5)
	EABORT       = syscall.Errno(__errno_base + 6)
	EOF          = syscall.Errno(__errno_base + 7)
	EBADF        = syscall.Errno(__errno_base + 8)
)

var (
	SUCCESS error = nil
)

type Poller interface {
	//Add(event *Event, mask int) error
	Modify(event *Event, mask int) error
	//Del(event *Event, mask int) error
	Wait(changes map[int]*Event, evs []*Event, timeout time.Duration) (n int, err error)
	Close() error
}

type EventLoop struct {
	Index  int
	poll   *Poll
	events map[int]*Event
	count  int32 // connection count
	Packet []byte
	WaitFor EngineHandler
}

func NewLoop(index int) (*EventLoop, error) {
	poll, err := OpenPoll(index)
	if err != nil {
		return nil, err
	}
	ep := &EventLoop{
		poll:   poll,
		events: make(map[int]*Event),
		count:  0,
		Index:  index,
		Packet: make([]byte, 0xFFFF),
	}
	return ep, nil
}

func (loop *EventLoop) GetPoll() *Poll {
	return loop.poll
}

func (loop *EventLoop) Detach(ev *Event) {
	loop.Watch(ev, AE_DEL|AE_READABLE|AE_WRITABLE)
	delete(loop.events, ev.Fd)
	loop.count -= 1
}

func (loop *EventLoop) CloseSocket(ev *Event) {
	//fmt.Printf("close[%d]\n", ev.Fd)
	//SetsockoptLinger(ev.Fd, 2)
	loop.Detach(ev)
	Close(ev.Fd)
}

func (loop *EventLoop) GetEvent(fd socket_t) *Event {
	ev, ok := loop.events[fd]
	if ok {
		return ev
	} else {
		return nil
	}
}

func (loop *EventLoop) Watch(event *Event, mask int) error {
	//if event & AE_ADD > 0 {
	loop.events[event.Fd] = event
	//} else if event & AE_DEL > 0 {
	//	delete(loop.events, event.Fd)
	//}
	err := loop.poll.Modify(event, mask)
	if err == nil {
		if (event.mask == 0) {
			delete(loop.events, event.Fd)
			//loop.CloseSocket(ev)
		}
	}
	return err
}

func (loop *EventLoop) StartLoop() (error) {
	const (
		maxWaitEventsBegin = 1 << 10 // 1024
		maxWaitEventsStop  = 1 << 15 // 32768
	)
	fmt.Println("start: ", loop.Index)
	//nextTicker := time.Now()
	var poll Poller
	poll = loop.poll
	//fmt.Println("3.1")
	var events = make([]*Event, maxWaitEventsBegin)
	for loop.WaitFor(loop) {
		//fmt.Println("3.1.1")
		delay := time.Duration(0)
		pn, err := poll.Wait(loop.events, events, delay)
		//fmt.Printf("pn = %d, event=%+v\n", pn, err)
		if err != nil && err != syscall.EINTR {
			return err
		}
		//fmt.Println("3.1.3")
		if pn < 1 {
			continue
		}
		//fmt.Println("wait: ", loop.Index)
		//fmt.Println("----------------")
		for i := 0; i < pn; i++ {
			//fmt.Println(">>>>>>>>>>>>>>>>>>>>")
			//fd, mask := poll.VerifyEvent(events, i)
			fd := events[i].Fd
			mask := events[i].mask
			ev, ok := loop.events[fd]
			//fmt.Printf("i = %d, fd = %d, mask = %d, event=%+v\n", i, fd, mask, ev)
			err = nil
			if ok {
				if mask & AE_ERROR > 0 {
					//loop.CloseSocket(ev)
					if ev.Close != nil {
						ev.Close(loop, ev)
					} else {
						loop.CloseSocket(ev)
					}
					continue
				}
				if mask & AE_READABLE > 0 {
					if ev.Accept != nil {
						//fmt.Println("accept: ", loop.Index)
						for {
							nerr := ev.Accept(loop, ev)
							if nerr == nil {
								//ev.Context = &InputStream{}
								loop.count += 1
								//break
							} else if nerr == syscall.EWOULDBLOCK || nerr == syscall.EAGAIN {
								break
							}
						}
					} else {
						//fmt.Println("read-1")
						err = ev.Read(loop, ev)
						//fmt.Println("read-2")
					}
				}
				if mask & AE_WRITABLE > 0 && ev.Write != nil {
					//fmt.Println("write")
					err = ev.Write(loop, ev)
				}
				if err == EOF || err == ERROR {
					//fmt.Println("close")
					if ev.Close != nil {
						ev.Close(loop, ev)
					} else {
						loop.CloseSocket(ev)
					}
				}
			}
			//fmt.Println(err)
			//fmt.Println("<<<<<<<<<<<<<<<<<<<<")
		} // end for
		if pn == len(events) && pn * 2 <= maxWaitEventsStop {
			events = make([]*Event, pn * 2)
		}
	}

	return nil
}
