package datadog

import (
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestClient(t *testing.T) {
	client := NewClient(DefaultAddress)

	for i := 0; i != 1000; i++ {
		client.HandleMeasures(time.Time{}, stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				{Name: "count", Value: stats.ValueOf(5)},
				{Name: "rtt", Value: stats.ValueOf(100 * time.Millisecond)},
			},
			Tags: []stats.Tag{
				{"answer", "42"},
				{"hello", "world"},
			},
		})
	}

	if err := client.Close(); err != nil {
		t.Error(err)
	}
}

func BenchmarkClient(b *testing.B) {
	for _, N := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("write a batch of %d measures to a client", N), func(b *testing.B) {
			client := NewClientWith(ClientConfig{
				Address:    DefaultAddress,
				BufferSize: MaxBufferSize,
			})

			measures := make([]stats.Measure, N)

			for i := range measures {
				measures[i] = stats.Measure{
					Name: "benchmark.test.metric",
					Fields: []stats.Field{
						{Name: "value", Value: stats.ValueOf(42)},
					},
					Tags: []stats.Tag{
						{"answer", "42"},
						{"hello", "world"},
					},
				}
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					client.HandleMeasures(time.Time{}, measures...)
				}
			})
		})
	}
}
