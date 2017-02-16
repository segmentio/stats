package netstats

import (
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/netx"
	"github.com/segmentio/stats"
)

type handler struct {
	metrics []stats.Metric
}

func (h *handler) HandleMetric(m *stats.Metric) {
	c := *m
	c.Tags = append([]stats.Tag{}, m.Tags...)
	c.Time = time.Time{} // discard because it's unpredicatable
	h.metrics = append(h.metrics, c)
}

func TestBaseConn(t *testing.T) {
	c1 := &testConn{}
	c2 := &conn{Conn: c1}

	if base := netx.BaseConn(c2); base != c1 {
		t.Error("bad base:", base)
	}
}

func TestConn(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine("netstats.test")
	e.Register(h)

	c := &testConn{}
	conn := NewConnWith(e, c)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	if !reflect.DeepEqual(h.metrics, []stats.Metric{
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.open.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.write.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     12,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.read.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     12,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.close.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestConnError(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine("netstats.test")
	e.Register(h)

	now := time.Now()

	c := &testConn{err: errTest}
	conn := NewConnWith(e, c)
	conn.SetDeadline(now)
	conn.SetReadDeadline(now)
	conn.SetWriteDeadline(now)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Read(make([]byte, 32))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	if !reflect.DeepEqual(h.metrics, []stats.Metric{
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.open.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "set-deadline"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "set-read-deadline"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "set-write-deadline"}},
			Value:     1,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.write.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "write"}},
			Value:     1,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.read.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "read"}},
			Value:     1,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.read.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "read"}},
			Value:     1,
		},
		{
			Type:      stats.HistogramType,
			Namespace: "netstats.test",
			Name:      "conn.read.bytes",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "read"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.error.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}, {"operation", "close"}},
			Value:     1,
		},
		{
			Type:      stats.CounterType,
			Namespace: "netstats.test",
			Name:      "conn.close.count",
			Tags:      []stats.Tag{{"protocol", "tcp"}},
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
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
