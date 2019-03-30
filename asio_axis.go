package asio

type Engine struct {
	AfterEvent EventHandler
}

// 启动一个server
func StartListen(numLoops int, handler DataHandler) {
	// 回调
	callback := &Callback{
		OnAccept: handler,
	}

	opened := func(loop *EventLoop) bool {
		ev := &Event{
			NumLoops: numLoops,
			Accept:   DefaultAccept,
			Context:  callback,
		}
		handler(callback)
		loop.Watch(ev, AE_ADD|AE_READABLE)
		return true
	}
	CreateEngine(numLoops, opened)
}
