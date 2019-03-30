// +build windows

package asio

import "syscall"

type Poll struct {
	index  int
	handle syscall.Handle
}

func OpenPoll(index int) (*Poll, error) {
	poll := Poll{handle: syscall.InvalidHandle}
	h, err := syscall.CreateIoCompletionPort(syscall.InvalidHandle, 0, 0, 0xffffffff)
	if err != nil {
		panic(err)
	}

	poll.index = index
	poll.handle = h

	return poll, err
}
