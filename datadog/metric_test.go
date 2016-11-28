package datadog

import "github.com/segmentio/stats"

var metrics = []struct {
	s string
	m stats.Metric
}{
	{
		s: "test.metric.small:0|c\n",
		m: stats.Metric{
			Type:   stats.CounterType,
			Name:   "test.metric.small",
			Tags:   nil,
			Value:  0,
			Sample: 1,
		},
	},

	{
		s: "test.metric.common:1|c|#hello:world,answer:42\n",
		m: stats.Metric{
			Type:   stats.CounterType,
			Name:   "test.metric.common",
			Tags:   []stats.Tag{{"hello", "world"}, {"answer", "42"}},
			Value:  1,
			Sample: 1,
		},
	},

	{
		s: "test.metric.large:1.234|c|@0.1|#hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world,hello:world\n",
		m: stats.Metric{
			Type: stats.CounterType,
			Name: "test.metric.large",
			Tags: []stats.Tag{
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
				{"hello", "world"},
			},
			Value:  1.234,
			Sample: 10,
		},
	},

	{
		s: "page.views:1|c\n",
		m: stats.Metric{
			Type:   stats.CounterType,
			Name:   "page.views",
			Value:  1,
			Sample: 1,
			Tags:   nil,
		},
	},

	{
		s: "fuel.level:0.5|g\n",
		m: stats.Metric{
			Type:   stats.GaugeType,
			Name:   "fuel.level",
			Value:  0.5,
			Sample: 1,
			Tags:   nil,
		},
	},

	{
		s: "song.length:240|h|@0.5\n",
		m: stats.Metric{
			Type:   stats.HistogramType,
			Name:   "song.length",
			Value:  240,
			Sample: 2,
			Tags:   nil,
		},
	},

	{
		s: "users.uniques:1234|h\n",
		m: stats.Metric{
			Type:   stats.HistogramType,
			Name:   "users.uniques",
			Value:  1234,
			Sample: 1,
			Tags:   nil,
		},
	},

	{
		s: "users.online:1|c|#country:china\n",
		m: stats.Metric{
			Type:   stats.CounterType,
			Name:   "users.online",
			Value:  1,
			Sample: 1,
			Tags:   []stats.Tag{{"country", "china"}},
		},
	},

	{
		s: "users.online:1|c|@0.5|#country:china\n",
		m: stats.Metric{
			Type:   stats.CounterType,
			Name:   "users.online",
			Value:  1,
			Sample: 2,
			Tags:   []stats.Tag{{"country", "china"}},
		},
	},
}
