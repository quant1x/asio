package http

import (
	"github.com/mymmsc/asio/util"
	"github.com/mymmsc/gox/errors"
	"syscall"
)

const (
	// Carriage-Return Line-Feed
	CR        = byte(13) // Carriage-Return \r
	LF        = byte(10) // Line-Feed \n
	CBUFFSIZE = 8192
)

// http编解码状态
type HttpAction int

const (
	OP_NONE     HttpAction = iota
	OP_HEADER              // 获取http-header
	OP_LENGTH              // 获取http-header Content-Length
	OP_BODY                // 读取http-body
	OP_FINISHED            // HTTP协议解析完毕
)

type Http struct {
	op             HttpAction // 当前操作阶段
	method         []byte
	version        []byte
	path           []byte
	charset        []byte // 字符集
	gotheader      bool
	keepalive      bool
	chunked        bool
	length         int64           // Content-Length value used for keep-alive
	cbuff          [CBUFFSIZE]byte // a buffer to store server response header
	read           int64           // amount of bytes read
	bread          int64           // amount of body read
	rwrite, rwrote int64           // keep pointers in what we write - across, EAGAINs
	stream         *util.Stream
}

type HttpRequest struct {
	Http
}

type HttpResponse struct {
	Http
}

func NewHttp() *Http {
	return &Http{
		op:        OP_NONE,
		gotheader: false,
		keepalive: false,
		chunked:   false,
		length:    0,
		read:      0,
		bread:     0,
		rwrote:    0,
		rwrite:    0,
		stream:    nil,
	}
}

func (h *Http) ReadLine(packet []byte) (data []byte, leftover []byte, err error) {
	datalen := len(packet)
	leftover = packet
	err = syscall.EAGAIN
	for i := 1; i < datalen; i++ {
		if packet[i] == LF {
			leftover = packet[i+1:]
			if packet[i-1] == CR {
				data = packet[:i-1]
				err = nil
			} else {
				data = nil
				err = errors.Errorf("error line")
			}
			break
		}
	}
	return data, leftover, err
}

func (h *Http) checkHeader(packet []byte) {

}

func (h *Http) Decode(packet []byte) {
	// 检查packet有效性
	if len(packet) == 0 {
		return
	}
	// 将packet添加缓冲区尾部
	data := h.stream.Begin(packet)
	if h.op == OP_NONE {
		h.op = OP_HEADER
	}

	switch h.op {
	case OP_HEADER: // 解析http-header
	case OP_LENGTH: // 确定content-length
	case OP_BODY: // 读取http-body
	}

	// 检查 头部 解析
	if !h.gotheader {
		// header域没有解析完成
		// 解析
		// 判断是否header域结束
	}
	// 检查 协议长度

	// 检查 body 解析
	// 判断是否chunked编码
	// chunked解码
	// 修正 stream 缓冲区
	h.stream.End(data)
}

func (h *Http) ReadHeader(s string) {
	//r, err := http.Post(s)
}
