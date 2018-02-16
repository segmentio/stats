package netstats

import (
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestBaseConn(t *testing.T) {
	c1 := &testConn{}
	c2 := &conn{Conn: c1}

	if base := c2.BaseConn(); base != c1 {
		t.Error("bad base:", base)
	}
}

func TestConn(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("netstats.test", h)

	c := &testConn{}
	conn := NewConnWith(e, c)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	expected := []stats.Measure{
		{
			Name: "netstats.test.conn.open",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{stats.T("protocol", "tcp")},
		},
		{
			Name: "netstats.test.conn.write",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
				stats.MakeField("bytes", 12, stats.Histogram),
			},
			Tags: []stats.Tag{stats.T("protocol", "tcp")},
		},
		{
			Name: "netstats.test.conn.read",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
				stats.MakeField("bytes", 12, stats.Histogram),
			},
			Tags: []stats.Tag{stats.T("protocol", "tcp")},
		},
		{
			Name: "netstats.test.conn.close",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{stats.T("protocol", "tcp")},
		},
	}

	if !reflect.DeepEqual(expected, h.Measures()) {
		t.Error("bad measures:")
		t.Logf("expected: %v", expected)
		t.Logf("found:    %v", h.Measures())
	}
}

func TestConnError(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("netstats.test", h)

	now := time.Now()

	c := &testConn{err: errTest}
	conn := NewConnWith(e, c)
	conn.SetDeadline(now)
	conn.SetReadDeadline(now)
	conn.SetWriteDeadline(now)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	expected := []stats.Measure{
		{
			Name: "netstats.test.conn.open",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{stats.T("protocol", "tcp")},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "set-deadline"),
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "set-read-deadline"),
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "set-write-deadline"),
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.write",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
				stats.MakeField("bytes", 0, stats.Histogram),
			},
			Tags: []stats.Tag{
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "write"),
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.read",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
				stats.MakeField("bytes", 0, stats.Histogram),
			},
			Tags: []stats.Tag{
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "read"),
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.close",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("protocol", "tcp"),
			},
		},
		{
			Name: "netstats.test.conn.error",
			Fields: []stats.Field{
				stats.MakeField("count", 1, stats.Counter),
			},
			Tags: []stats.Tag{
				stats.T("operation", "close"),
				stats.T("protocol", "tcp"),
			},
		},
	}

	if !reflect.DeepEqual(expected, h.Measures()) {
		t.Error("bad measures:")
		t.Logf("expected: %v", expected)
		t.Logf("found:    %v", h.Measures())
	}
}

func TestRootError(t *testing.T) {
	e1 := &net.OpError{Err: io.EOF}
	e2 := rootError(e1)

	if e2 != io.EOF {
		t.Errorf("bad root error: %s", e2)
	}
}

type testConn struct {
	bytes.Buffer
	err error
}

func (c *testConn) Read(b []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.Buffer.Read(b)
}

func (c *testConn) Write(b []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.Buffer.Write(b)
}

func (c *testConn) Close() error                       { return c.err }
func (c *testConn) LocalAddr() net.Addr                { return testLocalAddr }
func (c *testConn) RemoteAddr() net.Addr               { return testRemoteAddr }
func (c *testConn) SetDeadline(_ time.Time) error      { return c.err }
func (c *testConn) SetReadDeadline(_ time.Time) error  { return c.err }
func (c *testConn) SetWriteDeadline(_ time.Time) error { return c.err }

var (
	testLocalAddr  = &net.TCPAddr{IP: net.IP{127, 0, 0, 1}, Port: 2121}
	testRemoteAddr = &net.TCPAddr{IP: net.IP{127, 0, 0, 1}, Port: 4242}
	errTest        = errors.New("test")
)
