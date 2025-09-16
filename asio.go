package asio

import "errors"

var (
	//ErrMaxActiveConnReached 连接池超限
	ErrMaxActiveConnReached = errors.New("max active conn reached")
	// ErrClosed 连接已关闭
	ErrClosed = errors.New("pool is closed")
	// ErrIsNil 连接无效
	ErrIsNil = errors.New("connection is nil. rejecting")
)

// ConnectionPool 基本方法
type ConnectionPool interface {
	// Acquire 获取一个连接
	Acquire() (any, error)
	// Release 归还一个连接
	Release(any) error
	// CloseConn 关闭连接
	CloseConn(any) error
	// CloseAll 关闭全部连接
	CloseAll()
	// Close 关闭连接池, 释放所有连接
	Close() error
	// Len 连接数
	Len() int
}
