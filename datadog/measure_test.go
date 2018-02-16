package datadog

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

var (
	testMeasures = []struct {
		m stats.Measure
		s string
	}{
		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					stats.MakeField("count", 5, stats.Counter),
				},
			},
			s: `request.count:5|c
`,
		},

		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					stats.MakeField("count", 5, stats.Counter),
					stats.MakeField("rtt", 100*time.Millisecond, stats.Histogram),
				},
				Tags: []stats.Tag{
					stats.T("answer", "42"),
					stats.T("hello", "world"),
				},
			},
			s: `request.count:5|c|#answer:42,hello:world
request.rtt:0.1|h|#answer:42,hello:world
`,
		},
	}
)

func TestAppendMeasure(t *testing.T) {
	for _, test := range testMeasures {
		t.Run(test.s, func(t *testing.T) {
			if s := string(AppendMeasure(nil, test.m)); s != test.s {
				t.Error("bad metric representation:")
				t.Log("expected:", test.s)
				t.Log("found:   ", s)
			}
		})
	}
}
