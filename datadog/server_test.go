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

	addr, closer := startTestServer(t, HandlerFunc(func(m stats.Metric, _ net.Addr) {
		switch m.Name {
		case "test.A":
			atomic.AddUint32(&a, uint32(m.Value))

		case "test.B":
			atomic.AddUint32(&b, uint32(m.Value))

		case "test.C":
			atomic.AddUint32(&c, uint32(m.Value))
		}
	}))
	defer closer.Close()

	client := NewClient(ClientConfig{
		Address:       addr,
		Engine:        engine,
		FlushInterval: time.Millisecond,
	})
	defer client.Close()

	ma := stats.MakeCounter(engine, "test.A")
	ma.Incr()

	mb := stats.MakeCounter(engine, "test.B")
	mb.Incr()
	mb.Incr()

	mc := stats.MakeCounter(engine, "test.C")
	mc.Incr()
	mc.Incr()
	mc.Incr()

	time.Sleep(10 * time.Millisecond)

	if atomic.LoadUint32(&a) != 1 {
		t.Error("test.A not reported")
	}

	if atomic.LoadUint32(&b) != 2 {
		t.Error("test.B not reported")
	}

	if atomic.LoadUint32(&c) != 3 {
		t.Error("test.C not reported")
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
