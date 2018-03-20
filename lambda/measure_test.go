package lambda

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

var (
	timestamp   = time.Date(2018, 3, 16, 10, 0, 0, 123456789, time.UTC) // 1521194400123456789
	testMetrics = []struct {
		m stats.Measure
		s string
	}{
		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					{Name: "count", Value: stats.ValueOf(42)},
				},
			},
			s: `MONITORING|1521194400|42|count|request.count`,
		},
		{
			m: stats.Measure{
				Name: "cache",
				Fields: []stats.Field{
					{Name: "hit", Value: stats.ValueOf(10)},
					{Name: "miss", Value: stats.ValueOf(5)},
				},
				Tags: []stats.Tag{
					stats.T("bob", "alice"),
					stats.T("foo", "bar"),
				},
			},
			s: `MONITORING|1521194400|10|count|cache.hit|#bob=alice,foo=bar
MONITORING|1521194400|5|count|cache.miss|#bob=alice,foo=bar`,
		},
		{
			m: stats.Measure{
				Name: "user",
				Fields: []stats.Field{
					{Value: stats.ValueOf(10)},
				},
			},
			s: `MONITORING|1521194400|10|count|user`,
		},
	}
)

func TestAppendMeasure(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.s, func(t *testing.T) {
			if s := string(AppendMeasure(nil, timestamp, test.m)); s != (test.s + "\n") {
				t.Error("bad metric representation")
				t.Log("expected:", test.s)
				t.Log("got:", s)
			}
		})
	}
}
