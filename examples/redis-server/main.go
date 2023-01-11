package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mymmsc/asio"
	"github.com/mymmsc/gox/redis"
)

var (
	mu   sync.RWMutex
	keys = make(map[string]string)
)

type conn struct {
	is   asio.InputStream
	addr string
}

func accept(loop *asio.EventLoop, ev *asio.Event) error {
	nfd, rsa, err := asio.Accept(ev.Fd)
	if err != nil {
		return err
	}
	nev := &asio.Event{
		Fd:      nfd,
		Addr:    rsa,
		Read:    read,
		Write:   write,
		Context: &asio.InputStream{},
	}
	loop.Watch(nev, asio.AE_ADD|asio.AE_READABLE)

	return nil
}

func read(loop *asio.EventLoop, ev *asio.Event) error {
	n, nerr := asio.Recv(ev.Fd, loop.Packet)
	if nerr == nil || nerr == asio.SUCCESS {
		is := ev.Context.(*asio.InputStream)
		data := is.Begin(loop.Packet[0:n])
		var complete bool
		var err error
		var args [][]byte
		var out []byte
		for {
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
				switch strings.ToUpper(string(args[0])) {
				default:
					out = redis.AppendError(out, "ERR unknown command '"+string(args[0])+"'")
				case "PING":
					if len(args) > 2 {
						out = redis.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else if len(args) == 2 {
						out = redis.AppendBulk(out, args[1])
					} else {
						out = redis.AppendString(out, "PONG")
					}
				case "GET":
					if len(args) != 2 {
						out = redis.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						key := string(args[1])
						mu.Lock()
						val, ok := keys[key]
						mu.Unlock()
						if !ok {
							out = redis.AppendNull(out)
						} else {
							out = redis.AppendBulkString(out, val)
						}
					}
				case "SET":
					if len(args) != 3 {
						out = redis.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						key, val := string(args[1]), string(args[2])
						mu.Lock()
						keys[key] = val
						mu.Unlock()
						out = redis.AppendString(out, "OK")
					}
				case "DEL":
					if len(args) < 2 {
						out = redis.AppendError(out, "ERR wrong number of arguments for '"+string(args[0])+"' command")
					} else {
						var n int
						mu.Lock()
						for i := 1; i < len(args); i++ {
							if _, ok := keys[string(args[i])]; ok {
								n++
								delete(keys, string(args[i]))
							}
						}
						mu.Unlock()
						out = redis.AppendInt(out, int64(n))
					}
				case "FLUSHDB":
					mu.Lock()
					keys = make(map[string]string)
					mu.Unlock()
					out = redis.AppendString(out, "OK")
				}
				for {
					n, err = asio.Send(ev.Fd, out)
					if err == nil || err == asio.SUCCESS {
						err = nil
						break
					} else if err == asio.EAGAIN {
						continue
					} else if n == 0 {
						err = asio.ERROR
						break
					}
				}
			}
		}
		is.End(data)
		/*if len(out) > 0 {
			ev.Out = append([]byte{}, out...)
		}
		loop.Watch(ev, asio.AE_ADD|asio.AE_WRITABLE)
		loop.Watch(ev, asio.AE_DEL|asio.AE_READABLE)*/
	} else if nerr == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		//loop.CloseSocket(ev)
		nerr = asio.ERROR
	}
	return nerr
}

func write(loop *asio.EventLoop, ev *asio.Event) error {
	n, err := asio.Send(ev.Fd, ev.Out)
	if err == nil || err == asio.SUCCESS {
		//
	} else if err == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_WRITABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		err = asio.ERROR
	}
	if n == len(ev.Out) {
		ev.Out = nil
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
		loop.Watch(ev, asio.AE_DEL|asio.AE_WRITABLE)
	} else {
		ev.Out = ev.Out[n:]
	}
	return err
}

// multi loop
func main() {
	numLoops := 1
	flag.IntVar(&numLoops, "loops", 0, "num loops")
	flag.Parse()

	handle := func(loop *asio.EventLoop) bool {
		fd, err := asio.Listen("tcp://:9037")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		ev := &asio.Event{
			NumLoops: numLoops,
			Fd:       fd,
			Accept:   accept,
		}
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
		return true
	}
	asio.CreateEngine(numLoops, handle)
}
