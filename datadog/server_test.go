package datadog

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestServer(t *testing.T) {
	engine := stats.NewDefaultEngine()

	a := uint32(0)
	b := uint32(0)
	c := uint32(0)

	addr, closer := startTestServer(t, HandlerFunc(func(m Metric, _ net.Addr) {
		switch m.Name {
		case "datadog.test.A":
			atomic.AddUint32(&a, uint32(m.Value))

		case "datadog.test.B":
			atomic.AddUint32(&b, uint32(m.Value))

		case "datadog.test.C":
			atomic.AddUint32(&c, uint32(m.Value))

		default:
			t.Error("unexpected metric:", m)
		}
	}))
	defer closer.Close()

	client := NewClient(ClientConfig{
		Address:       addr,
		Engine:        engine,
		FlushInterval: time.Millisecond,
	})
	defer client.Close()

	ma := stats.MakeCounter(engine, "A")
	ma.Incr()
	ma.Incr()

	// Test that the counter properly computes the delta between flushes of the
	// client.
	time.Sleep(10 * time.Millisecond)
	ma.Incr()

	mb := stats.MakeGauge(engine, "B")
	mb.Set(1)
	mb.Set(2)
	mb.Set(3)

	mc := stats.MakeHistogram(engine, "C")
	mc.Observe(1)
	mc.Observe(2)
	mc.Observe(3)

	time.Sleep(10 * time.Millisecond)

	if n := atomic.LoadUint32(&a); n != 3 { // two increments (+1, +1, +1)
		t.Error("datadog.test.A: bad count:", n)
	}

	if n := atomic.LoadUint32(&b); n != 3 { // three assignments (=1, =2, =3)
		t.Error("datadog.test.B: bad count:", n)
	}

	if n := atomic.LoadUint32(&c); n != 2 { // observed values are averaged ((1 + 2 + 3) / 3)
		t.Error("datadog.test.C: bad count:", n)
	}
}

func startTestServer(t *testing.T, handler Handler) (addr string, closer io.Closer) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	go Serve(conn, handler)

	return conn.LocalAddr().String(), conn
}
