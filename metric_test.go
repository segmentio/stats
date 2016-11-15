package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestMetricKey(t *testing.T) {
	tests := []struct {
		key  string
		name string
		tags []Tag
	}{
		{
			key:  "?",
			name: "",
			tags: nil,
		},
		{
			key:  "M?",
			name: "M",
			tags: nil,
		},
		{
			key:  "M?A=1",
			name: "M",
			tags: []Tag{{"A", "1"}},
		},
		{
			key:  "M?A=1&B=2",
			name: "M",
			tags: []Tag{{"A", "1"}, {"B", "2"}},
		},
		{
			key:  "M?A=1&B=2&C=3",
			name: "M",
			tags: []Tag{{"A", "1"}, {"B", "2"}, {"C", "3"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if key := metricKey(test.name, test.tags); key != test.key {
				t.Errorf("metricKey(%#v, %#v) => %#v != %#v", test.name, test.tags, key, test.key)
			}
		})
	}
}

func TestSortMetrics(t *testing.T) {
	tests := []struct {
		metrics []Metric
	}{
		{
			metrics: nil,
		},
		{
			metrics: []Metric{
				Metric{Key: "X?"},
				Metric{Key: "M?A=1&B=2"},
			},
		},
		{
			metrics: []Metric{
				Metric{Key: "M?A=1&B=2"},
				Metric{Key: "X?"},
			},
		},
	}

	for _, test := range tests {
		sortMetrics(test.metrics)
		key := ""

		for _, m := range test.metrics {
			if m.Key < key {
				t.Errorf("sorting metrics did not produced an order sequence: %#v < %#v", m.Key, key)
				return
			}
			key = m.Key
		}
	}
}

func TestMetricStore(t *testing.T) {
	now := time.Now()

	store := makeMetricStore(metricStoreConfig{
		timeout: 10 * time.Millisecond,
	})

	// Push a couple of metrics to the store.
	store.apply(metricOp{
		typ:   CounterType,
		key:   "M?A=1&B=2",
		name:  "M",
		tags:  []Tag{{"A", "1"}, {"B", "2"}},
		value: 1,
		apply: metricOpAdd,
	}, now)

	store.apply(metricOp{
		typ:   CounterType,
		key:   "M?A=1&B=2",
		name:  "M",
		tags:  []Tag{{"A", "1"}, {"B", "2"}},
		value: 1,
		apply: metricOpAdd,
	}, now)

	store.apply(metricOp{
		typ:   CounterType,
		key:   "X?",
		name:  "X",
		tags:  nil,
		value: 10,
		apply: metricOpAdd,
	}, now.Add(5*time.Millisecond))

	// Check the state of the store.
	state := store.state()
	sortMetrics(state)

	if !reflect.DeepEqual(state, []Metric{
		Metric{
			Type:    CounterType,
			Key:     "M?A=1&B=2",
			Name:    "M",
			Tags:    []Tag{{"A", "1"}, {"B", "2"}},
			Value:   2,
			Version: 2,
		},
		Metric{
			Type:    CounterType,
			Key:     "X?",
			Name:    "X",
			Tags:    nil,
			Value:   10,
			Version: 1,
		},
	}) {
		t.Error("bad metric store state:", state)
	}

	// Expire metrics.
	store.deleteExpiredMetrics(now.Add(12 * time.Millisecond))

	// Check the state of the store after expiring metrics.
	state = store.state()
	sortMetrics(state)

	if !reflect.DeepEqual(state, []Metric{
		Metric{
			Type:    CounterType,
			Key:     "X?",
			Name:    "X",
			Tags:    nil,
			Value:   10,
			Version: 1,
		},
	}) {
		t.Error("bad metric store state:", state)
	}
}

func TestMetricDiff(t *testing.T) {
	tests := []struct {
		old, new, changed, unchanged, expired []Metric
	}{
		{
			old:       nil,
			new:       nil,
			changed:   nil,
			unchanged: nil,
			expired:   nil,
		},
		{
			old: []Metric{
				Metric{Key: "A?", Version: 1},
				Metric{Key: "A?hello=world", Version: 2},
				Metric{Key: "B?", Version: 1},
				Metric{Key: "C?", Version: 1},
			},
			new: []Metric{
				Metric{Key: "A?", Version: 1},
				Metric{Key: "B?", Version: 2},
				Metric{Key: "C?", Version: 3},
			},
			changed: []Metric{
				Metric{Key: "B?", Version: 2},
				Metric{Key: "C?", Version: 3},
			},
			unchanged: []Metric{
				Metric{Key: "A?", Version: 1},
			},
			expired: []Metric{
				Metric{Key: "A?hello=world", Version: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			changed, unchanged, expired := Diff(test.old, test.new)

			if !reflect.DeepEqual(changed, test.changed) {
				t.Errorf("changed: %#v != %#v", changed, test.changed)
			}

			if !reflect.DeepEqual(unchanged, test.unchanged) {
				t.Errorf("unchanged: %#v != %#v", unchanged, test.unchanged)
			}

			if !reflect.DeepEqual(expired, test.expired) {
				t.Errorf("expired: %#v != %#v", expired, test.expired)
			}
		})
	}
}
