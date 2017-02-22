package datadog

import (
	"net"
	"testing"

	"github.com/segmentio/stats"
)

func BenchmarkClient(b *testing.B) {
	addr, closer := startTestServer(nil, HandlerFunc(func(m Metric, a net.Addr) {}))
	defer closer.Close()

	engine := stats.NewEngine("datadog.test")
	engine.Register(NewClient(addr))

	b.Run("Add", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			engine.Add("A", float64(i))
		}
		engine.Flush()
	})

	b.Run("Set", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			engine.Set("B", float64(i))
		}
		engine.Flush()
	})

	b.Run("Observe", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			engine.Observe("C", float64(i))
		}
		engine.Flush()
	})
}
