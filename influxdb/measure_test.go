package influxdb

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

var (
	timestamp   = time.Date(2017, 7, 23, 3, 36, 0, 123456789, time.UTC)
	testMetrics = []struct {
		m stats.Measure
		s string
	}{
		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					{Name: "count", Value: stats.ValueOf(5)},
				},
			},
			s: `request count=5 1500780960123456789`,
		},

		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					{Name: "count", Value: stats.ValueOf(5)},
					{Name: "rtt", Value: stats.ValueOf(100 * time.Millisecond)},
				},
				Tags: []stats.Tag{
					stats.T("answer", "42"),
					stats.T("hello", "world"),
				},
			},
			s: `request,answer=42,hello=world count=5,rtt=0.1 1500780960123456789`,
		},
	}
)

func TestAppendMetric(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.s, func(t *testing.T) {
			if s := string(AppendMeasure(nil, timestamp, test.m)); s != (test.s + "\n") {
				t.Error("bad metric representation:")
				t.Log("expected:", test.s)
				t.Log("found:   ", s)
			}
		})
	}
}
