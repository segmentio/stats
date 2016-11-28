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
	changes []stats.Metric
}{
	{
		name:    "empty",
		old:     nil,
		new:     nil,
		changes: []stats.Metric{},
	},
	{
		name: "small",
		old: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "A?", Sample: 1},            // unchanged
			stats.Metric{Type: stats.CounterType, Key: "A?hello=world", Sample: 2}, // expired
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 1},            // changed
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 1},              // changed
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#0", Value: 0.1, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H2?", Key: "H2?#1", Value: 0.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#2", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#3", Value: 1.0, Sample: 1},
		},
		new: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "A?", Sample: 1},
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 2},
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 3},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#0", Value: 0.1, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H2?", Key: "H2?#1", Value: 0.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#1", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#2", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#3", Value: 1.0, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#4", Value: 0.5, Sample: 1},
			stats.Metric{Type: stats.HistogramType, Group: "H1?", Key: "H1?#5", Value: 1.0, Sample: 1},
		},
		changes: []stats.Metric{
			stats.Metric{Type: stats.CounterType, Key: "B?", Sample: 0},
			stats.Metric{Type: stats.GaugeType, Key: "C?", Sample: 0},
			stats.Metric{Type: stats.HistogramType, Key: "H1?", Value: 0.75, Sample: 2},
		},
	},
}

func TestDiff(t *testing.T) {
	for _, test := range diffs {
		t.Run(test.name, func(t *testing.T) {
			_, changes := diff(test.old, test.new)

			if !reflect.DeepEqual(changes, test.changes) {
				t.Errorf("\n<<< %#v\n>>> %#v", test.changes, changes)
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
