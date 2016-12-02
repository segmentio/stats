package netstats

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/segmentio/stats"
)

func NewListener(eng *stats.Engine, lstn net.Listener, tags ...stats.Tag) net.Listener {
	t0 := make([]stats.Tag, 0, len(tags)+1)
	t1 := append(t0, tags...)
	t2 := append(t1, stats.Tag{Name: "protocol", Value: lstn.Addr().Network()})

	return &listener{
		Listener: lstn,
		eng:      eng,
		tags:     t1,
		errors:   stats.MakeCounter(eng, "lstn.errors.count", t2...),
	}
}

type listener struct {
	net.Listener
	eng    *stats.Engine
	tags   []stats.Tag
	closed uint32
	errors stats.Counter
	once   sync.Once
}

func (lstn *listener) Accept() (conn net.Conn, err error) {
	if conn, err = lstn.Listener.Accept(); err != nil {
		if atomic.LoadUint32(&lstn.closed) == 0 {
			lstn.error("accept", err)
		}
	}

	if conn != nil {
		conn = NewConn(lstn.eng, conn, lstn.tags...)
	}

	return
}

func (lstn *listener) Close() (err error) {
	lstn.once.Do(func() {
		atomic.StoreUint32(&lstn.closed, 1)

		if err = lstn.Listener.Close(); err != nil {
			lstn.error("close", err)
		}
	})
	return
}

func (lstn *listener) error(op string, err error) {
	if e, ok := err.(net.Error); !ok || !(e.Temporary() || e.Timeout()) {
		lstn.errors.Clone(stats.Tag{Name: "operation", Value: op}).Incr()
	}
}
