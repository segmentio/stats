package netstats

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

type ConnTag int

const (
	TagProtocol   ConnTag = 1 << 0
	TagLocalAddr  ConnTag = 1 << 1
	TagLocalPort  ConnTag = 1 << 2
	TagRemoteAddr ConnTag = 1 << 3
	TagRemotePort ConnTag = 1 << 4

	TagAll = TagProtocol | TagLocalAddr | TagLocalPort | TagRemoteAddr | TagRemotePort
)

func NewConnTags(c net.Conn, t ConnTag) (tags stats.Tags) {
	tags = make(stats.Tags, 0, 5)

	laddr := c.LocalAddr()
	raddr := c.RemoteAddr()

	lhost, lport, _ := net.SplitHostPort(laddr.String())
	rhost, rport, _ := net.SplitHostPort(raddr.String())

	if (t & TagProtocol) != 0 {
		tags = append(tags, stats.Tag{"protocol", laddr.Network()})
	}

	if (t & TagLocalAddr) != 0 {
		tags = append(tags, stats.Tag{"local_addr", lhost})
	}

	if (t & TagLocalPort) != 0 {
		tags = append(tags, stats.Tag{"local_port", lport})
	}

	if (t & TagRemoteAddr) != 0 {
		tags = append(tags, stats.Tag{"remote_addr", rhost})
	}

	if (t & TagRemotePort) != 0 {
		tags = append(tags, stats.Tag{"remote_port", rport})
	}

	return
}

func NewConn(c net.Conn, client stats.Client, tags ...stats.Tag) net.Conn {
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
	err = c.Conn.Close()
	c.once.Do(func() {
		if err != nil {
			c.error("close", err)
		}
		c.metrics.Close.Add(1)
	})
	return
}

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
