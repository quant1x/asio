package v2

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func socket_opt(s rawSocket) {
	// 设置为非阻塞模式
	var mode uint32 = 1
	if err := windows.SetHandleInformation(s, windows.HANDLE_FLAG_INHERIT, mode); err != nil {
		panic(err)
	}
	// 设置超时时间（以毫秒为单位）
	timeout := uint32(5000) // 5秒
	err := windows.SetsockoptInt(s, windows.SOL_SOCKET, windows.SO_RCVTIMEO, int(timeout))
	if err != nil {
		fmt.Println("SetsockoptInt failed:", err)
		return
	}
}
