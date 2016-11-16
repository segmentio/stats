package datadog

import (
	"reflect"
	"testing"

	"github.com/segmentio/stats"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		old     []stats.Metric
		new     []stats.Metric
		changed []stats.Metric
	}{
		{
			old:     nil,
			new:     nil,
			changed: nil,
		},
		{
			old: []stats.Metric{
				stats.Metric{Key: "A?", Sample: 1},            // unchanged
				stats.Metric{Key: "A?hello=world", Sample: 2}, // expired
				stats.Metric{Key: "B?", Sample: 1},            // changed
				stats.Metric{Key: "C?", Sample: 1},            // changed
			},
			new: []stats.Metric{
				stats.Metric{Key: "A?", Sample: 1},
				stats.Metric{Key: "B?", Sample: 2},
				stats.Metric{Key: "C?", Sample: 3},
			},
			changed: []stats.Metric{
				stats.Metric{Key: "B?", Sample: 0},
				stats.Metric{Key: "C?", Sample: 0},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			_, changed := diff(test.old, test.new, map[string]stats.Metric{}, nil)

			if !reflect.DeepEqual(changed, test.changed) {
				t.Errorf("changed: %#v != %#v", changed, test.changed)
			}
		})
	}
}
