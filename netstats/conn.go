package netstats

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

// NewConn returns a net.Conn object that wraps c and produces metrics on eng.
func NewConn(eng *stats.Engine, c net.Conn, tags ...stats.Tag) net.Conn {
	t0 := make([]stats.Tag, 0, len(tags)+2)
	t0 = append(t0, tags...)
	t0 = append(t0, stats.Tag{Name: "protocol", Value: c.LocalAddr().Network()})

	nc := &conn{
		Conn:   c,
		close:  stats.MakeCounter(eng, "conn.close.count", t0...),
		errors: stats.MakeCounter(eng, "conn.errors.count", t0...),
	}

	t1 := append(t0, stats.Tag{Name: "operation", Value: "read"})
	nc.reads = stats.MakeHistogram(eng, "conn.iops", t1...)
	nc.bytesIn = stats.MakeCounter(eng, "conn.bytes.count", t1...)

	t2 := append(t0, stats.Tag{Name: "operation", Value: "write"})
	nc.writes = stats.MakeHistogram(eng, "conn.iops", t2...)
	nc.bytesOut = stats.MakeCounter(eng, "conn.bytes.count", t2...)

	stats.MakeCounter(eng, "conn.open.count", t0...).Incr()
	return nc
}

type conn struct {
	net.Conn
	once     sync.Once
	close    stats.Counter
	reads    stats.Histogram
	writes   stats.Histogram
	bytesIn  stats.Counter
	bytesOut stats.Counter
	errors   stats.Counter
}

func (c *conn) Close() (err error) {
	err = c.Conn.Close()
	c.once.Do(func() {
		if err != nil {
			c.error("close", err)
		}
		c.close.Incr()
	})
	return
}

func (c *conn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 {
		c.reads.Observe(float64(n))
		c.bytesIn.Add(float64(n))
	}

	if err != nil && err != io.EOF {
		c.error("read", err)
	}

	return
}

func (c *conn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n > 0 {
		c.writes.Observe(float64(n))
		c.bytesOut.Add(float64(n))
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
			c.errors.Clone(stats.Tag{Name: "operation", Value: op}).Incr()
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
