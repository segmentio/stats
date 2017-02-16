package netstats

import (
	"io"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/segmentio/netx"
	"github.com/segmentio/stats"
)

const (
	protocol  = "protocol"
	operation = "operation"
)

// NewConn returns a net.Conn object that wraps c and produces metrics on eng.
//
// If eng is nil, the default engine is used.
func NewConn(eng *stats.Engine, c net.Conn) net.Conn {
	if eng == nil {
		eng = stats.DefaultEngine
	}

	nc := &conn{
		Conn:  c,
		eng:   eng,
		proto: c.LocalAddr().Network(),
	}
	runtime.SetFinalizer(nc, func(c *conn) { c.Close() })

	eng.Incr("conn.open.count", stats.T(protocol, nc.proto))
	return nc
}

type conn struct {
	net.Conn
	eng   *stats.Engine
	proto string
	once  sync.Once
}

func (c *conn) BaseConn() net.Conn {
	return c.Conn
}

func (c *conn) Close() (err error) {
	err = c.Conn.Close()
	c.once.Do(func() {
		if err != nil {
			c.error("close", err)
		}
		c.eng.Incr("conn.close.count", stats.T(protocol, c.proto))
	})
	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n >= 0 {
		c.eng.Observe("conn.read.bytes", float64(n), stats.T(protocol, c.proto))
	}

	if err != nil && err != io.EOF {
		c.error("read", err)
	}

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n >= 0 {
		c.eng.Observe("conn.write.bytes", float64(n), stats.T(protocol, c.proto))
	}

	if err != nil {
		c.error("write", err)
	}

	return
}

func (c *conn) SetDeadline(t time.Time) (err error) {
	if err = c.Conn.SetDeadline(t); err != nil {
		c.error("set-deadline", err)
	}
	return
}

func (c *conn) SetReadDeadline(t time.Time) (err error) {
	if err = c.Conn.SetReadDeadline(t); err != nil {
		c.error("set-read-deadline", err)
	}
	return
}

func (c *conn) SetWriteDeadline(t time.Time) (err error) {
	if err = c.Conn.SetWriteDeadline(t); err != nil {
		c.error("set-write-deadline", err)
	}
	return
}

func (c *conn) error(op string, err error) {
	switch err = rootError(err); err {
	case io.EOF, io.ErrClosedPipe, io.ErrUnexpectedEOF:
		// this is expected to happen when connections are closed
	default:
		// only report serious errors, others should be handled gracefully
		if !netx.IsTemporary(err) {
			c.eng.Incr("conn.error.count", stats.T(protocol, c.proto), stats.T(operation, op))
		}
	}
}

func rootError(err error) error {
searchRootError:
	for i := 0; i != 10; i++ { // protect against cyclic errors
		switch e := err.(type) {
		case *net.OpError:
			err = e.Err
		default:
			break searchRootError
		}
	}
	return err
}
