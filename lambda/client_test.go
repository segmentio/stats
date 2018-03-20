package lambda

import (
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func BenchmarkClient(b *testing.B) {
	for _, N := range []int{1, 10, 110} {
		b.Run(fmt.Sprintf("write %d measures", N), func(b *testing.B) {
			client := NewClient()

			timestamp := time.Now()
			measures := make([]stats.Measure, N)

			for i := range measures {
				measures[i] = stats.Measure{
					Name: "benchmark.cache",
					Fields: []stats.Field{
						{Name: "hit", Value: stats.ValueOf(42)},
					},
					Tags: []stats.Tag{
						stats.T("bob", "alice"),
						stats.T("foo", "bar"),
					},
				}
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					client.HandleMeasures(timestamp, measures...)
				}
			})
		})
	}
}
