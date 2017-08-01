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
	engine := stats.NewEngine("datadog.test", nil)

	a := uint32(0)
	b := uint32(0)
	c := uint32(0)

	addr, closer := startTestServer(t, HandlerFunc(func(m Metric, _ net.Addr) {
		switch m.Name {
		case "datadog.test.A":
			atomic.AddUint32(&a, uint32(m.Value))

		case "datadog.test.B":
			atomic.StoreUint32(&b, uint32(m.Value)) // gauge

		case "datadog.test.C":
			atomic.AddUint32(&c, uint32(m.Value))

		default:
			t.Error("unexpected metric:", m)
		}
	}))
	defer closer.Close()

	client := NewClient(addr)
	defer client.Close()
	engine.Register(client)

	engine.Incr("A")
	engine.Incr("A")
	engine.Incr("A")

	engine.Set("B", 1)
	engine.Set("B", 2)
	engine.Set("B", 3)

	engine.Observe("C", 1)
	engine.Observe("C", 2)
	engine.Observe("C", 3)

	engine.Flush()

	// Give time for the server to receive the metrics.
	time.Sleep(100 * time.Millisecond)

	if n := atomic.LoadUint32(&a); n != 3 { // two increments (+1, +1, +1)
		t.Error("datadog.test.A: bad value:", n)
	}

	if n := atomic.LoadUint32(&b); n != 3 { // three assignments (=1, =2, =3)
		t.Error("datadog.test.B: bad value:", n)
	}

	if n := atomic.LoadUint32(&c); n != 6 { // observed values, all reported (+1, +2, +3)
		t.Error("datadog.test.C: bad value:", n)
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
