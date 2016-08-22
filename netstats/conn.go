package netstats

import (
	"io"
	"net"
	"time"

	"github.com/segmentio/stats"
)

func NewConn(c net.Conn, client stats.Client, tags ...stats.Tag) net.Conn {
	laddr := c.LocalAddr()
	raddr := c.RemoteAddr()
	tags = append(tags,
		stats.Tag{"protocol", laddr.Network()},
		stats.Tag{"local_address", laddr.String()},
		stats.Tag{"remote_address", raddr.String()},
	)
	return conn{
		Conn:    c,
		metrics: NewMetrics(client, tags...),
	}
}

type conn struct {
	net.Conn
	metrics *Metrics
}

func (c conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 {
		c.metrics.BytesIn.Observe(float64(n))
	}

	if err != nil {
		c.error("read", err)
	}

	return
}

func (c conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n > 0 {
		c.metrics.BytesOut.Observe(float64(n))
	}

	if err != nil {
		c.error("write", err)
	}

	return
}

func (c conn) Close() (err error) {
	if err = c.Conn.Close(); err != nil {
		c.error("close", err)
	}
	return
}

func (c conn) SetDeadline(t time.Time) (err error) {
	if err = c.Conn.SetDeadline(t); err != nil {
		c.error("set-timeout", err)
	}
	return
}

func (c conn) SetReadDeadline(t time.Time) (err error) {
	if err = c.Conn.SetReadDeadline(t); err != nil {
		c.error("set-read-timeout", err)
	}
	return
}

func (c conn) SetWriteDeadline(t time.Time) (err error) {
	if err = c.Conn.SetWriteDeadline(t); err != nil {
		c.error("set-write-timeout", err)
	}
	return
}

func (c conn) error(op string, err error) {
	// this is expected, don't report it
	if err != io.EOF {
		// only report serious errors, these should be handled gracefully
		if e, ok := err.(net.Error); !ok || !(e.Temporary() || e.Timeout()) {
			c.metrics.Errors.Add(1, stats.Tag{"operation", op})
		}
	}
}
