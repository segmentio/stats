package stats

import (
	"reflect"
	"sort"
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
		sort.Sort(NaturalMetricOrder(test.metrics))
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
	store.update(Metric{
		Type:  CounterType,
		Key:   "M?A=1&B=2",
		Name:  "M",
		Tags:  []Tag{{"A", "1"}, {"B", "2"}},
		Value: 1,
	}, now)

	store.update(Metric{
		Type:  CounterType,
		Key:   "M?A=1&B=2",
		Name:  "M",
		Tags:  []Tag{{"A", "1"}, {"B", "2"}},
		Value: 1,
	}, now)

	store.update(Metric{
		Type:  CounterType,
		Key:   "X?",
		Name:  "X",
		Tags:  nil,
		Value: 10,
	}, now.Add(5*time.Millisecond))

	// Check the state of the store.
	if state := store.state(); !reflect.DeepEqual(state, []Metric{
		Metric{
			Type:  CounterType,
			Key:   "M?A=1&B=2",
			Name:  "M",
			Tags:  []Tag{{"A", "1"}, {"B", "2"}},
			Value: 2,
		},
		Metric{
			Type:  CounterType,
			Key:   "X?",
			Name:  "X",
			Tags:  nil,
			Value: 10,
		},
	}) {
		t.Error("bad metric store state:", state)
	}

	// Expire metrics.
	store.deleteExpiredMetrics(now.Add(12 * time.Millisecond))

	// Check the state of the store after expiring metrics.
	if state := store.state(); !reflect.DeepEqual(state, []Metric{
		Metric{
			Type:  CounterType,
			Key:   "X?",
			Name:  "X",
			Tags:  nil,
			Value: 10,
		},
	}) {
		t.Error("bad metric store state:", state)
	}
}
