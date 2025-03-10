package datadog

import (
	"io"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/stats/v4"
	"github.com/segmentio/stats/v4/statstest"
)

func TestServer(t *testing.T) {
	statstest.DisableVersionReporting(t)

	engine := stats.NewEngine("datadog.test", nil)

	a := uint32(0)
	c := uint32(0)

	seenGauges := make([]Metric, 0)
	var mu sync.Mutex

	addr, closer := startTestServer(t, HandlerFunc(func(m Metric, _ net.Addr) {
		switch m.Name {
		case "datadog.test.A":
			atomic.AddUint32(&a, uint32(m.Value))

		case "datadog.test.B":
			// Because it's the other side of a HTTP server, these can arrive
			// out of order, even if the client sends them in the right order
			// - there aren't any guarantees about which connection the server
			// will activate first.
			//
			// Previously this used atomic.StoreInt32 to do last write wins, but
			// occasionally the last write would be "2" or "1" and fail the
			// test, easily reproducible by running this test 200 times.
			mu.Lock()
			seenGauges = append(seenGauges, m)
			mu.Unlock()

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

	now := time.Now()
	engine.Set("B", float64(time.Since(now)))
	engine.Set("B", float64(time.Since(now)))
	last := float64(time.Since(now))
	engine.Set("B", last)

	engine.Observe("C", 1)
	engine.Observe("C", 2)
	engine.Observe("C", 3)

	// because this is "last write wins" it's possible it runs before the reads
	// of 1 or 2; add a sleep to try to ensure it loses the race
	engine.Flush()

	// Give time for the server to receive the metrics.
	time.Sleep(20 * time.Millisecond)

	if n := atomic.LoadUint32(&a); n != 3 { // two increments (+1, +1, +1)
		t.Error("datadog.test.A: bad value:", n)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seenGauges) != 3 {
		t.Errorf("datadog.test.B: expected 3 values, got %d", len(seenGauges))
	}
	sort.Slice(seenGauges, func(i, j int) bool {
		return seenGauges[i].Value < seenGauges[j].Value
	})
	if seenGauges[2].Value != last {
		t.Errorf("expected highest value to be the latest value set, got %v", seenGauges[2])
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
