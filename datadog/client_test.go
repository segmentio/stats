package datadog

import (
	"reflect"
	"testing"

	"github.com/segmentio/stats"
)

var diffs = []struct {
	name    string
	old     []stats.Metric
	new     []stats.Metric
	changed []stats.Metric
}{
	{
		name:    "empty",
		old:     nil,
		new:     nil,
		changed: []stats.Metric{},
	},
	{
		name: "small",
		old: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "A?", Sample: 1},            // unchanged
			stats.Metric{Type: stats.CounterType, Key: "A?hello=world", Sample: 2}, // expired
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 1},            // changed
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 1},              // changed
			stats.Metric{Type: stats.HistogramType, Key: "H1?#0", Name: "H1", Value: 0.1, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H2?#1", Name: "H2", Value: 0.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#1", Name: "H1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#2", Name: "H1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#3", Name: "H1", Value: 1.0, Sample: 1},
		},
		new: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "A?", Sample: 1},
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 2},
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 3},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#0", Name: "H1", Value: 0.1, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H2?#1", Name: "H2", Value: 0.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#1", Name: "H1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#2", Name: "H1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#3", Name: "H1", Value: 1.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#4", Name: "H1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Key: "H1?#5", Name: "H1", Value: 1.0, Sample: 1},
		},
		changed: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 0},
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 0},
			stats.Metric{Type: stats.HistogramType, Name: "H1", Value: 0.75, Sample: 2},
		},
	},
}

func TestDiff(t *testing.T) {
	for _, test := range diffs {
		t.Run(test.name, func(t *testing.T) {
			_, changed := diff(test.old, test.new)

			if !reflect.DeepEqual(changed, test.changed) {
				t.Errorf("changed: %#v != %#v", changed, test.changed)
			}
		})
	}
}

func BenchmarkDiff(b *testing.B) {
	for _, test := range diffs {
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				diff(test.old, test.new)
			}
		})
	}
}
