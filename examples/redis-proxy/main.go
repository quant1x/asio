package main

import (
	"flag"
	"fmt"
	"github.com/mymmsc/asio"
	"os"
)

var (
	destHost = "127.0.0.1"
	destPort = 17170
	destAddr = fmt.Sprintf("%s:%d", destHost, destPort)
	connPool = make([]*RedisConnPool, 0)
)

// multi loop
func main() {
	numLoops := 1
	flag.IntVar(&numLoops, "loops", 1, "num loops")
	flag.Parse()

	cp := NewRedisConnPool(destAddr, REDIS_POOL_MAX)
	connPool = append(connPool, cp)

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
