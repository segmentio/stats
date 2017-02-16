package netstats

import (
	"net"
	"sync/atomic"

	"github.com/segmentio/netx"
	"github.com/segmentio/stats"
)

func NewListener(lstn net.Listener) net.Listener {
	return NewListenerEngine(nil, lstn)
}

func NewListenerEngine(eng *stats.Engine, lstn net.Listener) net.Listener {
	if eng == nil {
		eng = stats.DefaultEngine
	}
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
		conn = NewConn(l.eng, conn)
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
	if !netx.IsTemporary(err) {
		l.eng.Incr("conn.error.count",
			stats.T("protocol", l.Addr().Network()),
			stats.T("operation", op),
		)
	}
}
