package main

import (
	"errors"
	"github.com/mymmsc/asio"
	"github.com/mymmsc/goapi/redis"
	"strings"
)

func redisAuth0(v interface{}) error {
	var password string = "1234"
	fd := v.(int)
	var array redis.RespArray
	array = append(array, redis.BulkString("AUTH"))
	array = append(array, redis.BulkString(password))
	cmd := array.Protocol()
	reqBuf := []byte(cmd)
	n, err := asio.Send(fd, reqBuf)
	if err != nil {
		return err
	}
	var respBuf [1024]byte
	n, err = asio.Recv(fd, respBuf[:])
	if n < 1 || err != nil {
		return err
	}
	resp, err := redis.DecodeFromBytes(respBuf[:n])
	if err == nil {
		op, _ := resp.Op()
		k, _ := resp.Key()
		_ = op
		_ = k
		//log.Debugf("<<op[%s],key[%s]: %s", op, k, resp.Value)
		cmdResponse := string(resp.Value)
		if strings.EqualFold(cmdResponse, "OK") {
			return nil
		} else {
			return errors.New("ping failed")
		}
	} else if err != nil {
		return err
	}
	return nil
}

func redisAuth(v interface{}) error {
	var password string = "1234"
	fd := v.(int)
	var array redis.RespArray
	array = append(array, redis.BulkString("AUTH"))
	array = append(array, redis.BulkString(password))
	cmd := array.Protocol()
	reqBuf := []byte(cmd)
	return redisCommand(fd, reqBuf, func(reply []byte) error {
		cmdResponse := string(reply)
		if strings.EqualFold(cmdResponse, "OK") {
			return nil
		} else {
			return errors.New("AUTH failed")
		}
	})
}

func redisPing0(v interface{}) error {
	fd := v.(int)
	reqBuf := []byte("PING\r\n")
	n, err := asio.Send(fd, reqBuf)
	if err != nil {
		return err
	}
	var respBuf [1024]byte
	n, err = asio.Recv(fd, respBuf[0:1024])
	if n < 1 || err != nil {
		return err
	}
	resp, err := redis.DecodeFromBytes(respBuf[:n])
	if err == nil {
		op, _ := resp.Op()
		k, _ := resp.Key()
		_ = op
		_ = k
		//log.Debugf("<<op[%s],key[%s]: %s", op, k, resp.Value)
		cmdResponse := string(resp.Value)
		if strings.EqualFold(cmdResponse, "pong") {
			return nil
		} else {
			return errors.New("PING failed")
		}
	} else if err != nil {
		return err
	}
	return nil
}

func redisPing(v interface{}) error {
	fd := v.(int)
	reqBuf := []byte("PING\r\n")
	return redisCommand(fd, reqBuf, func(reply []byte) error {
		cmdResponse := string(reply)
		if strings.EqualFold(cmdResponse, "pong") {
			return nil
		} else {
			return errors.New("ping failed")
		}
	})
}

func redisCommand(fd int, cmd []byte, checkout func(reply []byte) error) (error) {
	//fmt.Printf("proxy to remote: request[%s]\n", cmd)
	n, err := asio.Send(fd, cmd)
	if err != nil {
		return err
	}
	var respBuf [1024]byte
	tolen := 0
	for {
		n, err = asio.Recv(fd, respBuf[tolen:])
		if n < 1 && err != asio.EAGAIN {
			return err
		}
		tolen += n
		var resp *redis.Resp
		resp, err = redis.DecodeFromBytes(respBuf[:tolen])
		if err == nil {
			/*op, _ := resp.Op()
			k, _ := resp.Key()
			_ = op
			_ = k*/
			//log.Debugf("<<op[%s],key[%s]: %s", op, k, resp.Value)
			//fmt.Printf("remote to proxy: response[%s]\n", resp.Value)
			err = checkout(resp.Value)
			break
		} else if err == asio.EAGAIN {
			continue
		} else if err != nil {
			continue
		}
	}
	return err
}
