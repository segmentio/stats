package netstats

import (
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestConn(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	c := &testConn{}
	conn := NewConn(engine, c)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	// Give time to the engine to process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics, _ := engine.State(0)
	sort.Sort(stats.MetricsByKey(metrics))

	expects := []stats.Metric{
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.bytes.count?operation=read&protocol=tcp",
			Name:  "conn.bytes.count",
			Tags:  []stats.Tag{{"operation", "read"}, {"protocol", "tcp"}},
			Value: 12,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.bytes.count?operation=write&protocol=tcp",
			Name:  "conn.bytes.count",
			Tags:  []stats.Tag{{"operation", "write"}, {"protocol", "tcp"}},
			Value: 12,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.close.count?protocol=tcp",
			Name:  "conn.close.count",
			Tags:  []stats.Tag{{"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.HistogramType,
			Key:   "conn.iops?operation=read&protocol=tcp",
			Name:  "conn.iops",
			Tags:  []stats.Tag{{"operation", "read"}, {"protocol", "tcp"}},
			Value: 12,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.HistogramType,
			Key:   "conn.iops?operation=write&protocol=tcp",
			Name:  "conn.iops",
			Tags:  []stats.Tag{{"operation", "write"}, {"protocol", "tcp"}},
			Value: 12,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.open.count?protocol=tcp",
			Name:  "conn.open.count",
			Tags:  []stats.Tag{{"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
	}

	for i := range metrics {
		metrics[i].Time = time.Time{} // reset because we can't predict that value
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad engine state:")

		for i := range metrics {
			m := metrics[i]
			e := expects[i]

			if !reflect.DeepEqual(m, e) {
				t.Logf("unexpected metric at index %d:\n<<< %#v\n>>> %#v", i, m, e)
			}
		}
	}
}

func TestConnError(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	now := time.Now()

	c := &testConn{err: errTest}
	conn := NewConn(engine, c)
	conn.SetDeadline(now)
	conn.SetReadDeadline(now)
	conn.SetWriteDeadline(now)
	conn.Write([]byte("Hello World!"))
	conn.Read(make([]byte, 32))
	conn.Read(make([]byte, 32))
	conn.Read(make([]byte, 32))
	conn.Close()
	conn.Close() // idempotent: only reported once

	// Give time to the engine to process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics, _ := engine.State(0)
	sort.Sort(stats.MetricsByKey(metrics))

	expects := []stats.Metric{
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.close.count?protocol=tcp",
			Name:  "conn.close.count",
			Tags:  []stats.Tag{{"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=close&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "close"}, {"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=read&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "read"}, {"protocol", "tcp"}},
			Value: 3,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=set-read-timeout&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "set-read-timeout"}, {"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=set-timeout&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "set-timeout"}, {"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=set-write-timeout&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "set-write-timeout"}, {"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.errors.count?operation=write&protocol=tcp",
			Name:  "conn.errors.count",
			Tags:  []stats.Tag{{"operation", "write"}, {"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
		stats.Metric{
			Type:  stats.CounterType,
			Key:   "conn.open.count?protocol=tcp",
			Name:  "conn.open.count",
			Tags:  []stats.Tag{{"protocol", "tcp"}},
			Value: 1,
			Namespace: stats.Namespace{
				Name: "netstats.test",
			},
		},
	}

	for i := range metrics {
		metrics[i].Time = time.Time{} // reset because we can't predict that value
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad engine state:")

		for i := range metrics {
			m := metrics[i]
			e := expects[i]

			if !reflect.DeepEqual(m, e) {
				t.Logf("unexpected metric at index %d:\n<<< %#v\n>>> %#v", i, m, e)
			}
		}
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
