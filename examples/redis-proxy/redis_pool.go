package main

import (
	"fmt"
	"github.com/mymmsc/asio"
	"github.com/mymmsc/goapi/pool"
	"time"
)

type RedisConnPool struct {
	addr string
	pool pool.Pool
}

const (
	REDIS_POOL_MAX = 20000
	REDIS_TIMEOUT  = 30000
)

func NewRedisConnPool(addr string, size int) *RedisConnPool {
	//ping 检测连接的方法
	//ping := redisPing
	//factory 创建连接的方法
	factory := func() (interface{}, error) {
		fd, err := asio.Socket()
		if err == nil {
			err = asio.Connect(fd, addr)
		}
		if err == nil {
			asio.Setsockopt(fd)
		}
		err = redisAuth(fd)
		if err != nil {
			//asio.Close(fd)
			//fmt.Println(err)
			// 忽略错误
			err = nil
		}
		return fd, err
	}

	//close 关闭连接的方法
	close := func(v interface{}) error {
		fd := v.(int)
		return asio.Close(fd)
	}

	//创建一个连接池： 初始化5，最大连接30
	poolConfig := &pool.PoolConfig{
		InitialCap: 5,
		MaxCap:     size,
		Factory:    factory,
		Close:      close,
		//Ping:       ping,
		//连接最大空闲时间，超过该时间的连接 将会关闭，可避免空闲时连接EOF，自动失效的问题
		IdleTimeout: 300 * time.Second,
	}
	pool, err := pool.NewChannelPool(poolConfig)
	if err != nil {
		fmt.Println("err=", err)
	}
	cp := &RedisConnPool{
		addr: addr,
		pool: pool,
	}
	return cp
}

func (p *RedisConnPool) GetConn() interface{} {
	conn, err := p.pool.Get()
	if err != nil {
		return nil
	}
	return conn
}

func (p *RedisConnPool) ReturnConn(conn interface{}) {
	p.pool.Put(conn)
}

/*
func (p *RedisConnPool) MarkUnusable(context net.Conn) {
	if pc, ok := context.(*group.PoolConn); ok {
		pc.MarkUnusable()
	}
}*/
