package asio

import (
	"syscall"
)

type EventHandler func(loop *EventLoop, ev *Event) error
type DataHandler func(cb *Callback) error

type Event struct {
	Fd       socket_t
	Addr     syscall.Sockaddr
	mask     int
	Out      []byte // write buffer
	Context  interface{}
	Accept   EventHandler
	Connect  EventHandler
	Read     EventHandler
	Write    EventHandler
	Timeout  EventHandler
	Close    EventHandler
	NumLoops int
}

type Callback struct {
	Owner     interface{}
	OnAccept  DataHandler
	OnConnect DataHandler
	Opened    func(addr syscall.Sockaddr)
	OnRead    func(cb *Callback, packet []byte) error
	OnWrite   func(ev *Callback) ([]byte, error)
	OnTimeout DataHandler
	Closed    DataHandler
}

func (e *Event) delegate(handler interface{}) (EventHandler, DataHandler) {
	var eh EventHandler = nil
	var dh DataHandler = nil
	var ok bool = false
	if handler != nil {
		if eh, ok = handler.(EventHandler); ok {
			//
		} else if dh, ok = handler.(DataHandler); ok {
			//
		}
	}
	if !ok {
		return nil, nil
	}

	return eh, dh
}

func DefaultAccept(loop *EventLoop, ev *Event) error {
	nfd, rsa, err := Accept(ev.Fd)
	if err != nil {
		return err
	}
	ncb := &Callback{}
	nev := &Event{
		Fd:      nfd,
		Addr:    rsa,
		Read:    DefaultRead,
		Write:   DefaultWrite,
		Close:   DefaultClose,
		Context: ncb,
	}
	loop.Watch(nev, AE_ADD|AE_READABLE)
	if ev.Context != nil {
		cb := ev.Context.(*Callback)
		if cb.OnAccept != nil {
			err = cb.OnAccept(ncb)
		}
	}
	return nil
}

func DefaultClose(loop *EventLoop, ev *Event) error {
	var err error = nil
	//owner := ev.Context

	if ev.Context != nil {
		cb := ev.Context.(*Callback)
		if cb.Closed != nil {
			err = cb.Closed(cb)
		}
	}
	loop.CloseSocket(ev)

	return err
}

func DefaultRead(loop *EventLoop, ev *Event) error {
	n, nerr := Recv(ev.Fd, loop.Packet[0:])
	if nerr == nil || nerr == SUCCESS {
		//in := loop.Packet[0:n]
		/*ctx := ev.Context.(*context)
		//fmt.Printf("remote->local: in[%s]\n", in)
		data := ctx.is.Begin(in)
		//fmt.Printf("remote->local: data[%s]\n", data)
		//fmt.Printf("local:%d, remote:%d, data length:%d \n", ctx.fd, ev.Fd, n)
		sent := 0
		for {
			sent, nerr = Send(ctx.fd, in)
			if nerr == nil || nerr == SUCCESS {
				nerr = nil
				//fmt.Printf("local:%d, remote:%d \n", ctx.fd, ev.Fd)
				break
			} else if nerr == EAGAIN {
				continue
			} else if sent == 0 {
				nerr = ERROR
				break
			}
		}
		data = data[sent:]
		ctx.is.End(data)*/
		if ev.Context != nil {
			cb := ev.Context.(*Callback)
			if cb.OnRead != nil {
				nerr = cb.OnRead(cb, loop.Packet[:n])
			}
		}
		loop.Watch(ev, AE_ADD|AE_WRITABLE)
	} else if nerr == EAGAIN {
		loop.Watch(ev, AE_ADD|AE_READABLE)
	} else if n == 0 /* || err == ERROR || err == EOF*/ {
		nerr = ERROR
	}
	return nerr
}

func DefaultWrite(loop *EventLoop, ev *Event) error {
	var nerr error
	var data []byte
	if ev.Context != nil {
		cb := ev.Context.(*Callback)
		if cb.OnWrite != nil {
			data, nerr = cb.OnWrite(cb)
		}
	}
	if len(data) > 0 {
		ev.Out = append(ev.Out, data...)
	}
	n, err := Send(ev.Fd, ev.Out)
	if nerr == EOF {
		err = nerr
	} else if err == nil || err == SUCCESS {
		//
	} else if err == EAGAIN {
		loop.Watch(ev, AE_ADD|AE_WRITABLE)
	} else if n == 0 /* || err == ERROR || err == EOF*/ {
		err = ERROR
	}
	if n == len(ev.Out) {
		ev.Out = nil
		loop.Watch(ev, AE_ADD|AE_READABLE)
		loop.Watch(ev, AE_DEL|AE_WRITABLE)
	} else {
		ev.Out = ev.Out[n:]
	}
	return err
}
