package http

import (
	"github.com/mymmsc/asio/util"
	"testing"
)

func reader(s string) *Http {
	is := util.NewStream([]byte(s))

	return &Http{stream: is}
}

func TestReadLine(t *testing.T) {
	str := "line1\r\nline2\r\n\r\n"
	packet := []byte(str)
	r := reader(str)
	s, l, err := r.ReadLine(packet)
	t.Logf("line = [%s]", s)
	if string(s) != "line1" || err != nil {
		t.Logf("Line 1: %s, %v", s, err)
	}
	s, l, err = r.ReadLine(l)
	t.Logf("line = [%s]", s)
	if string(s) != "line2" || err != nil {
		t.Fatalf("Line 2: %s, %v", s, err)
	}
	// 模拟http协议header结束的情况
	s, l, err = r.ReadLine(l)
	t.Logf("line = [%s]", s)
	if string(s) != "" || err != nil {
		t.Fatalf("Line 3: %s, %v", s, err)
	}
	_ = l
}
