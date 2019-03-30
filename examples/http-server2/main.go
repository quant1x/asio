package main

import (
	"flag"
	"fmt"
	"github.com/mymmsc/asio"
	"os"
	"strconv"
	"time"
)

func accept(cb *asio.Callback) (error) {
	cb.OnRead = read
	cb.OnWrite = write
	return nil
}

func read(ev *asio.Callback, packet []byte) (error) {
	return nil
}

func write(ev *asio.Callback) ([]byte, error) {
	var b []byte
	var result = appendresp(b, "200 OK", "", "Hello World!\r\n")

	return result, asio.EOF
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

		cb := &asio.Callback{
			OnAccept: accept,
		}

		ev := &asio.Event{
			NumLoops: numLoops,
			Fd:       fd,
			Accept:   asio.DefaultAccept,
			Context:  cb,
		}
		loop.Watch(ev, asio.AE_ADD|asio.AE_READABLE)

		return true
	}
	asio.CreateEngine(numLoops, handle)
}
