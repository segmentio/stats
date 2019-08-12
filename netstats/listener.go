package netstats

import (
	"net"
	"sync/atomic"

	"github.com/segmentio/stats"
)

func NewListener(lstn net.Listener) net.Listener {
	return NewListenerWith(stats.DefaultEngine, lstn)
}

func NewListenerWith(eng *stats.Engine, lstn net.Listener) net.Listener {
	return &listener{
		lstn: lstn,
		eng:  eng,
	}
}

type listener struct {
	lstn   net.Listener
	eng    *stats.Engine
	closed uint32
}

func (l *listener) Accept() (conn net.Conn, err error) {
	if conn, err = l.lstn.Accept(); err != nil {
		if atomic.LoadUint32(&l.closed) == 0 {
			l.error("accept", err)
		}
	}

	if conn != nil {
		conn = NewConnWith(l.eng, conn)
	}

	return
}

func (l *listener) Close() (err error) {
	atomic.StoreUint32(&l.closed, 1)
	return l.lstn.Close()
}

func (l *listener) Addr() net.Addr {
	return l.lstn.Addr()
}

func (l *listener) error(op string, err error) {
	if !isTemporary(err) {
		l.eng.Incr("conn.error.count",
			stats.T("operation", op),
			stats.T("protocol", l.Addr().Network()),
		)
	}
}

func isTemporary(err error) bool {
	e, ok := err.(interface {
		Temporary() bool
	})
	return ok && e.Temporary()
}
