package datadog

import (
	"testing"

	"github.com/segmentio/stats"
)

var testMetrics = []struct {
	s string
	m Metric
}{
	{
		s: "test.metric.small:0|c\n",
		m: Metric{
			Type:  Counter,
			Name:  "test.metric.small",
			Tags:  nil,
			Value: 0,
			Rate:  1,
		},
	},

	{
		s: "test.metric.common:1|c|#hello:world,answer:42\n",
		m: Metric{
			Type: Counter,
			Name: "test.metric.common",
			Tags: []stats.Tag{
				stats.T("hello", "world"),
				stats.T("answer", "42"),
			},
			Value: 1,
			Rate:  1,
		},
	},

	{
		s: "test.metric.large:1.234|c|@0.1|#hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world\n",
		m: Metric{
			Type: Counter,
			Name: "test.metric.large",
			Tags: []stats.Tag{
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
				stats.T("hello", "world"),
			},
			Value: 1.234,
			Rate:  0.1,
		},
	},

	{
		s: "page.views:1|c\n",
		m: Metric{
			Type:  Counter,
			Name:  "page.views",
			Value: 1,
			Rate:  1,
			Tags:  nil,
		},
	},

	{
		s: "fuel.level:0.5|g\n",
		m: Metric{
			Type:  Gauge,
			Name:  "fuel.level",
			Value: 0.5,
			Rate:  1,
			Tags:  nil,
		},
	},

	{
		s: "song.length:240|h|@0.5\n",
		m: Metric{
			Type:  Histogram,
			Name:  "song.length",
			Value: 240,
			Rate:  0.5,
			Tags:  nil,
		},
	},

	{
		s: "users.uniques:1234|h\n",
		m: Metric{
			Type:  Histogram,
			Name:  "users.uniques",
			Value: 1234,
			Rate:  1,
			Tags:  nil,
		},
	},

	{
		s: "users.online:1|c|#country:china\n",
		m: Metric{
			Type:  Counter,
			Name:  "users.online",
			Value: 1,
			Rate:  1,
			Tags: []stats.Tag{
				stats.T("country", "china"),
			},
		},
	},

	{
		s: "users.online:1|c|@0.5|#country:china\n",
		m: Metric{
			Type:  Counter,
			Name:  "users.online",
			Value: 1,
			Rate:  0.5,
			Tags: []stats.Tag{
				stats.T("country", "china"),
			},
		},
	},
}

func TestMetricString(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.s, func(t *testing.T) {
			if s := test.m.String(); s != test.s {
				t.Error(s)
			}
		})
	}
}
