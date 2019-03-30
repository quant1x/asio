package internal

/*
struct ngx_event_s {
    void            *data;
    unsigned         write:1;
    unsigned         accept:1;
    // used to detect the stale events in kqueue and epoll
	unsigned         instance:1;
	// the event was passed or would be passed to a kernel;
 	// in aio mode - operation was posted.
	unsigned         active:1;
	unsigned         disabled:1;
	// the ready event; in aio mode 0 means that no operation can be posted
	unsigned         ready:1;
	unsigned         oneshot:1;
	// aio operation is complete
	unsigned         complete:1;
	unsigned         eof:1;
	unsigned         error:1;
	unsigned         timedout:1;
	unsigned         timer_set:1;
	unsigned         delayed:1;
	unsigned         deferred_accept:1;
	// the pending eof reported by kqueue, epoll or in aio chain operation
	unsigned         pending_eof:1;
	unsigned         posted:1;
	unsigned         closed:1;
	// to test on worker exit
	unsigned         channel:1;
	unsigned         resolver:1;

	unsigned         cancelable:1;

	//#if (NGX_HAVE_KQUEUE)
	unsigned         kq_vnode:1;
	// the pending errno reported by kqueue
	int              kq_errno;
	//#endif

	// kqueue only:
	//   accept:     number of sockets that wait to be accepted
	//   read:       bytes to read when event is ready
	//               or lowat when event is set with NGX_LOWAT_EVENT flag
	//   write:      available space in buffer when event is ready
	//               or lowat when event is set with NGX_LOWAT_EVENT flag
	//
	// epoll with EPOLLRDHUP:
	//   accept:     1 if accept many, 0 otherwise
	//   read:       1 if there can be data to read, 0 otherwise
	//
	// iocp: TODO
	//
	// otherwise:
	//   accept:     1 if accept many, 0 otherwise
	//

	#if (NGX_HAVE_KQUEUE) || (NGX_HAVE_IOCP)
	int              available;
	#else
	unsigned         available:1;
	#endif

	ngx_event_handler_pt  handler;


	//#if (NGX_HAVE_IOCP)
	//ngx_event_ovlp_t ovlp;
	//#endif
};
 */
import "C"
