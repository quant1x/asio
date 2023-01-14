package main

import (
	"flag"
	"fmt"
	"github.com/mymmsc/asio"
	"os"
	"strconv"
	"time"
)

func accept(loop *asio.EventLoop, ev *asio.Event) error {
	nfd, rsa, err := asio.Accept(ev.Fd)
	if err != nil {
		return err
	}
	err = asio.Setsockopt(nfd)
	//err = syscall.SetNonblock(nfd, true)
	if err != nil {
		return err
	}

	nev := &asio.Event{
		Fd:    nfd,
		Addr:  rsa,
		Read:  read,
		Write: write,
	}
	loop.Watch(nev, asio.AE_ADD|asio.AE_READABLE)

	return nil
}

func read(loop *asio.EventLoop, ev *asio.Event) error {
	n, err := asio.Recv(ev.Fd, loop.Packet[0:])
	if err == nil || err == asio.SUCCESS {
		//loop.Watch(ev, asio.AE_DEL|asio.AE_READABLE)
		loop.Watch(ev, asio.AE_ADD|asio.AE_WRITABLE)
	} else if err == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		//loop.CloseSocket(ev)
		err = asio.ERROR
	}
	return err
}

func write(loop *asio.EventLoop, ev *asio.Event) error {
	var b []byte
	var result = appendresp(b, "200 OK", "", "Hello World!\r\n")
	n, err := asio.Send(ev.Fd, result)
	if err == nil || err == asio.SUCCESS {
		//loop.CloseSocket(ev)
		err = asio.EOF
	} else if err == asio.EAGAIN {
		loop.Watch(ev, asio.AE_ADD|asio.AE_WRITABLE)
	} else if n == 0 /* || err == asio.ERROR || err == asio.EOF*/ {
		err = asio.ERROR
	}
	return err
}

func appendresp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: asio\r\n"...)
	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	b = append(b, head...)
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// single loop
func main_single() {

	loop, err := asio.NewLoop(0)
	if err != nil {
		panic(err)
	}

	fd, err := asio.Listen("tcp://:8080")

	ev := &asio.Event{
		NumLoops: 2,
		Fd:       fd,
		Accept:   accept,
	}
	loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)
	loop.StartLoop()
}

// multi loop
func main() {
	numLoops := 1
	flag.IntVar(&numLoops, "loops", 1, "num loops")
	flag.Parse()

	handle := func(loop *asio.EventLoop) bool {
		fd, err := asio.Listen("tcp://:8080")
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
