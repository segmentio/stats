package datadog

import (
	"testing"
	"time"

	"github.com/vertoforce/stats"
)

var (
	testMeasures = []struct {
		m  stats.Measure
		s  string
		dp []string
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
			dp: []string{},
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
			dp: []string{},
		},

		{
			m: stats.Measure{
				Name: "request",
				Fields: []stats.Field{
					stats.MakeField("dist_rtt", 100*time.Millisecond, stats.Histogram),
				},
				Tags: []stats.Tag{
					stats.T("answer", "42"),
					stats.T("hello", "world"),
				},
			},
			s: `request.dist_rtt:0.1|d|#answer:42,hello:world
`,
			dp: []string{"dist_"},
		},
	}
)

func TestAppendMeasure(t *testing.T) {
	client := NewClient(DefaultAddress)
	for _, test := range testMeasures {
		t.Run(test.s, func(t *testing.T) {
			client.distPrefixes = test.dp
			if s := string(client.AppendMeasure(nil, test.m)); s != test.s {
				t.Error("bad metric representation:")
				t.Log("expected:", test.s)
				t.Log("found:   ", s)
			}
		})
	}
}

var (
	testDistNames = []struct {
		n string
		d bool
	}{
		{
			n: "name",
			d: false,
		},
		{
			n: "",
			d: false,
		},
		{
			n: "dist_name",
			d: true,
		},
		{
			n: "distname",
			d: false,
		},
	}
	distPrefixes = []string{"dist_"}
)

func TestSendDist(t *testing.T) {
	client := NewClientWith(ClientConfig{DistributionPrefixes: distPrefixes})
	for _, test := range testDistNames {
		t.Run(test.n, func(t *testing.T) {
			a := client.sendDist(test.n)
			if a != test.d {
				t.Error("distribution name detection incorrect:")
				t.Log("expected:", test.d)
				t.Log("found:   ", a)
			}
		})
	}
}
