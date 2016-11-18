package netstats

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

// NewConn returns a net.Conn object that wraps c and produces metrics on eng.
func NewConn(c net.Conn, eng *stats.Engine, tags ...stats.Tag) net.Conn {
	tags = append(tags, stats.Tag{Name: "protocol", Value: c.LocalAddr().Network()})
	nc := &conn{
		Conn: c,
		metrics: metrics{
			open:     eng.Counter("conn.open.count", tags...),
			close:    eng.Counter("conn.close.count", tags...),
			reads:    eng.Histogram("conn.iops", append(tags, stats.Tag{Name: "operation", Value: "read"})...),
			writes:   eng.Histogram("conn.iops", append(tags, stats.Tag{Name: "operation", Value: "write"})...),
			bytesIn:  eng.Counter("conn.bytes.count", append(tags, stats.Tag{Name: "operation", Value: "read"})...),
			bytesOut: eng.Counter("conn.bytes.count", append(tags, stats.Tag{Name: "operation", Value: "write"})...),
			errors:   eng.Counter("conn.errors.count", tags...),
		},
	}
	nc.metrics.open.Incr()
	return nc
}

type metrics struct {
	open     stats.Counter
	close    stats.Counter
	reads    stats.Histogram
	writes   stats.Histogram
	bytesIn  stats.Counter
	bytesOut stats.Counter
	errors   stats.Counter
}

type conn struct {
	net.Conn
	metrics
	once sync.Once
}

func (c *conn) Close() (err error) {
	err = c.Conn.Close()
	c.once.Do(func() {
		if err != nil {
			c.error("close", err)
		}
		c.metrics.close.Incr()
	})
	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 {
		c.metrics.reads.Observe(float64(n))
		c.metrics.bytesIn.Add(float64(n))
	}

	if err != nil && err != io.EOF {
		c.error("read", err)
	}

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n > 0 {
		c.metrics.writes.Observe(float64(n))
		c.metrics.bytesOut.Add(float64(n))
	}

	if err != nil {
		c.error("write", err)
	}

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
			c.metrics.errors.Clone(stats.Tag{Name: "operation", Value: op}).Incr()
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
