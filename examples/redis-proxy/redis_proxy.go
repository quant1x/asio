package main

import (
	"sync"

	"github.com/mymmsc/asio"
	"github.com/mymmsc/goapi/redis"
)

var (
	mu   sync.RWMutex
	keys = make(map[string]string)

	tmpLen = 1024 * 64
	tmpBuf = make([]byte, tmpLen)
)

const (
	PROXY_UNKNOWN = 0x00
	PROXY_LOCAL   = PROXY_UNKNOWN + 1
	PROXY_REMOTE  = PROXY_UNKNOWN + 2
)

type context struct {
	is    asio.InputStream
	addr  string
	side  int
	fd    int
	group int // 分组
}

func accept(loop *asio.EventLoop, ev *asio.Event) (error) {
	nfd, rsa, err := asio.Accept(ev.Fd)
	if err != nil {
		return err
	}

	nev := &asio.Event{
		Fd:      nfd,
		Addr:    rsa,
		Read:    read_local,
		Write:   nil,
		Close:   close,
		Context: &context{side: PROXY_LOCAL, group: asio.INVALID},
	}
	loop.Watch(nev, asio.AE_ADD|asio.AE_READABLE)

	return nil
}

func close(loop *asio.EventLoop, ev *asio.Event) (error) {
	ctx := ev.Context.(*context)
	if ctx.side == PROXY_LOCAL {
		lev := ev
		loop.CloseSocket(lev)
		rev := loop.GetEvent(ctx.fd)
		loop.Detach(rev)
		cp := connPool[ctx.group]
		cp.ReturnConn(ctx.fd)
	} else if ctx.side == PROXY_REMOTE {
		lev := loop.GetEvent(ctx.fd)
		loop.CloseSocket(lev)

		rev := ev
		loop.Detach(rev)
		cp := connPool[ctx.group]
		cp.ReturnConn(ctx.fd)
	}

	return nil
}

func read_local(loop *asio.EventLoop, ev *asio.Event) (error) {
	n, nerr := asio.Recv(ev.Fd, loop.Packet)
	if nerr == nil || nerr == asio.SUCCESS {
		ctx := ev.Context.(*context)
		in := loop.Packet[0:n]
		data := ctx.is.Begin(in)
		var complete bool
		var err error
		var args [][]byte
		var out []byte
		for {
			//fmt.Printf("local->remote: %s\n", data)
			complete, args, _, data, err = redis.ReadNextCommand(data, args[:0])
			if err != nil {
				nerr = asio.ERROR
				out = redis.AppendError(out, err.Error())
				break
			}
			if !complete {
				break
			}
			if len(args) > 0 {
				var dest interface{}
				var remote_change bool = true
				//op := string(args[0])
				k := args[0]
				if len(args) > 1 {
					k = args[1]
				}
				//_ = op
				hash := int(crc16(k)) % int(len(connPool))
				i := ctx.group
				if i == asio.INVALID {
					i = 0
					cp := connPool[i]
					dest = cp.GetConn()
				} else if i != hash {
					connPool[i].ReturnConn(ctx.fd)
					rev := loop.GetEvent(ctx.fd)
					loop.Detach(rev)
					i = hash
					cp := connPool[i]
					dest = cp.GetConn()
				} else {
					dest = ctx.fd
					remote_change = false
				}
				ctx.group = i
				remote := dest.(int)
				ctx.fd = remote
				ctxRemote := &context{
					side:  PROXY_REMOTE,
					fd:    ev.Fd,
					group: i,
				}
				//fmt.Printf("local->remote: command[%s]\n", in)
				for {
					n, err = asio.Send(remote, in)
					if err == nil || err == asio.SUCCESS {
						err = nil
						if remote_change {
							nev := &asio.Event{
								Fd:      remote,
								Addr:    nil,
								Read:    read_remote,
								Write:   nil,
								Close:   close,
								Context: ctxRemote,
							}
							loop.Watch(nev, asio.AE_ADD|asio.AE_READABLE)
						}
						//fmt.Printf("local:%d, remote:%d \n", ev.Fd, remote)
						break
					} else if err == asio.EAGAIN {
						continue
					} else if n == 0 {
						err = asio.ERROR
						break
					}
				}
				break
			}
		}
		ctx.is.End(data)
	} else if nerr == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		//loop.CloseSocket(ev)
		nerr = asio.ERROR
	}
	return nerr
}

func read_remote(loop *asio.EventLoop, ev *asio.Event) (error) {
	n, nerr := asio.Recv(ev.Fd, loop.Packet)
	if nerr == nil || nerr == asio.SUCCESS {
		ctx := ev.Context.(*context)
		in := loop.Packet[0:n]
		//fmt.Printf("remote->local: in[%s]\n", in)
		data := ctx.is.Begin(in)
		//fmt.Printf("remote->local: data[%s]\n", data)
		//fmt.Printf("local:%d, remote:%d, data length:%d \n", ctx.fd, ev.Fd, n)
		sent := 0
		for {
			sent, nerr = asio.Send(ctx.fd, in)
			if nerr == nil || nerr == asio.SUCCESS {
				nerr = nil
				//fmt.Printf("local:%d, remote:%d \n", ctx.fd, ev.Fd)
				break
			} else if nerr == asio.EAGAIN {
				continue
			} else if sent == 0 {
				nerr = asio.ERROR
				break
			}
		}
		data = data[sent:]
		ctx.is.End(data)
	} else if nerr == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		//loop.CloseSocket(ev)
		nerr = asio.ERROR
	}
	return nerr
}
