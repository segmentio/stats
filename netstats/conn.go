package netstats

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

func NewConn(c net.Conn, client stats.Client, tags ...stats.Tag) net.Conn {
	laddr := c.LocalAddr()
	raddr := c.RemoteAddr()

	lhost, lport, _ := net.SplitHostPort(laddr.String())
	rhost, rport, _ := net.SplitHostPort(raddr.String())

	tags = append(tags,
		stats.Tag{"protocol", laddr.Network()},
		stats.Tag{"local_addr", lhost},
		stats.Tag{"local_port", lport},
		stats.Tag{"remote_addr", rhost},
		stats.Tag{"remote_port", rport},
	)

	m := NewMetrics(client, tags...)
	m.Open.Add(1)

	return &conn{Conn: c, metrics: m}
}

type conn struct {
	net.Conn
	metrics *Metrics
	once    sync.Once
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 {
		c.metrics.Reads.Observe(float64(n))
		c.metrics.BytesIn.Add(float64(n))
	}

	if err != nil && err != io.EOF {
		c.error("read", err)
	}

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n > 0 {
		c.metrics.Writes.Observe(float64(n))
		c.metrics.BytesOut.Add(float64(n))
	}

	if err != nil {
		c.error("write", err)
	}

	return
}

func (c *conn) Close() (err error) {
	if err = c.Conn.Close(); err != nil {
		c.error("close", err)
	}
	c.once.Do(c.close)
	return
}

func (c *conn) close() { c.metrics.Close.Add(1) }

func (c *conn) SetDeadline(t time.Time) (err error) {
	if err = c.Conn.SetDeadline(t); err != nil {
		c.error("set-timeout", err)
	}
	return
}

func (c *conn) SetReadDeadline(t time.Time) (err error) {
	if err = c.Conn.SetReadDeadline(t); err != nil {
		c.error("set-read-timeout", err)
	}
	return
}

func (c *conn) SetWriteDeadline(t time.Time) (err error) {
	if err = c.Conn.SetWriteDeadline(t); err != nil {
		c.error("set-write-timeout", err)
	}
	return
}

func (c *conn) error(op string, err error) {
	switch err = rootError(err); err {
	case io.EOF, io.ErrClosedPipe, io.ErrUnexpectedEOF:
		// this is expected to happen when connections are closed
	default:
		// only report serious errors, others should be handled gracefully
		if e, ok := err.(net.Error); !ok || !(e.Temporary() || e.Timeout()) {
			c.metrics.Errors.Add(1, stats.Tag{"operation", op})
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
