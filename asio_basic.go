package asio

import (
	"runtime"
	"sync"
)

// LoadBalance sets the load balancing method.
type LoadBalance int

const (
	// Random requests that connections are randomly distributed.
	Random LoadBalance = iota
	// RoundRobin requests that connections are distributed to a loop in a
	// round-robin fashion.
	RoundRobin
	// LeastConnections assigns the next accepted connection to the loop with
	// the least number of active connections.
	LeastConnections
)

type engine struct {
	numLoops int
	loops    []*EventLoop   // all the loops
	wg       sync.WaitGroup // loop close waitgroup
	cond     *sync.Cond     // shutdown signaler
	balance  LoadBalance    // load balancing method
	accepted uintptr        // accept counter
}

type EngineHandler func(loop *EventLoop) bool

func CreateEngine(numLoops int, engineHandler EngineHandler) (*engine, error) {
	// 计算出要使用的loops/Goroutine的正确数目
	//numLoops := event.NumLoops
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}

	e := &engine{
		numLoops: numLoops,
		balance:  LeastConnections,
	}

	e.cond = sync.NewCond(&sync.Mutex{})

	defer func() {
		// wait on a signal for shutdown
		e.waitForShutdown()

		// notify all loops to close by closing all listeners
		//for _, l := range e.loops {
		//	l.poll.Trigger(errClosing)
		//}

		// wait on all loops to complete reading events
		e.wg.Wait()

		// close loops and all outstanding connections
		for _, loop := range e.loops {
			for _, c := range loop.events {
				loop.CloseSocket(c)
			}
			_ = loop.poll.Close()
		}
		//println("-- server stopped")
	}()

	// create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		loop, err := NewLoop(i)
		if err != nil {
			panic(err)
		}
		loop.WaitFor = e.waitForStop

		//l.poll.AddRead(ln.fd)
		if engineHandler(loop) {
			e.loops = append(e.loops, loop)
		}
	}
	// start loops in background
	e.wg.Add(len(e.loops))
	for _, loop := range e.loops {
		go loop.StartLoop()
	}

	return e, nil
}

// waitForShutdown waits for a signal to shutdown
func (s *engine) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins server closing
func (s *engine) signalShutdown() {
	s.cond.L.Lock()
	s.cond.Signal()
	s.cond.L.Unlock()
}

func (s *engine) waitForStop(loop *EventLoop) bool {
	return true
}
