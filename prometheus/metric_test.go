package prometheus

import (
	"math"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestMetricStore(t *testing.T) {
	input := []metric{
		{mtype: counter, scope: "test", name: "A", value: 1},
		{mtype: counter, scope: "test", name: "A", value: 2},
		{mtype: histogram, scope: "test", name: "C", value: 0.1},
		{mtype: gauge, scope: "test", name: "B", value: 1, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: counter, scope: "test", name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, scope: "test", name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: histogram, scope: "test", name: "C", value: 0.1},
		{mtype: gauge, scope: "test", name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: histogram, scope: "test", name: "C", value: 0.5},
		{mtype: histogram, scope: "test", name: "C", value: 10},
	}

	store := metricStore{}

	for _, m := range input {
		store.update(m, []stats.Value{
			stats.ValueOf(0.25),
			stats.ValueOf(0.5),
			stats.ValueOf(0.75),
			stats.ValueOf(1.0),
		})
	}

	metrics := store.collect(nil)
	sort.Sort(byNameAndLabels(metrics))

	expects := []metric{
		{mtype: counter, scope: "test", name: "A", value: 3, labels: labels{}},
		{mtype: counter, scope: "test", name: "A", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, scope: "test", name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: gauge, scope: "test", name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 2, labels: labels{{"le", "0.25"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "0.5"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "0.75"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "1"}}},
		{mtype: histogram, scope: "test", name: "C_count", value: 4, labels: labels{}},
		{mtype: histogram, scope: "test", name: "C_sum", value: 10.7, labels: labels{}},
	}

	if !reflect.DeepEqual(metrics, expects) {
		t.Error("bad metrics:")
		t.Logf("expected: %v", expects)
		t.Logf("found:    %v", metrics)
	}
}

func TestMetricEntryCleanup(t *testing.T) {
	now := time.Now()

	empty := false
	entry := metricEntry{
		mtype: counter,
		name:  "A",
		states: metricStateMap{
			0: []*metricState{
				&metricState{value: 42, time: now},
				&metricState{value: 1, time: now.Add(-time.Minute)},
				&metricState{value: 2, time: now.Add(-(500 * time.Millisecond))},
			},
			1: []*metricState{
				&metricState{value: 123, time: now.Add(10 * time.Millisecond)},
			},
			2: []*metricState{},
		},
	}

	callback := func() { empty = true }

	// Cleanup all states older than 1 second.
	entry.cleanup(now.Add(-time.Second), callback)

	if empty {
		t.Error("unexpected call to notify that the entry is empty")
	}

	if !reflect.DeepEqual(entry.states, metricStateMap{
		0: []*metricState{
			&metricState{value: 42, time: now},
			&metricState{value: 2, time: now.Add(-(500 * time.Millisecond))},
		},
		1: []*metricState{
			&metricState{value: 123, time: now.Add(10 * time.Millisecond)},
		},
	}) {
		t.Errorf("bad entry states: %#v", entry.states)
	}

	// Cleanup all states older than now to check that the comparison is
	// inclusive.
	entry.cleanup(now, callback)

	if empty {
		t.Error("unexpected call to notify that the entry is empty")
	}

	if !reflect.DeepEqual(entry.states, metricStateMap{
		1: []*metricState{
			&metricState{value: 123, time: now.Add(10 * time.Millisecond)},
		},
	}) {
		t.Errorf("bad entry states: %#v", entry.states)
	}

	// Cleanup all states.
	entry.cleanup(now.Add(time.Second), callback)

	if !empty {
		t.Error("callback not called!")
	}

	if !reflect.DeepEqual(entry.states, metricStateMap{}) {
		t.Errorf("bad entry states: %#v", entry.states)
	}
}

func TestMetricStoreCleanup(t *testing.T) {
	now := time.Now()

	store := metricStore{}
	store.update(metric{mtype: counter, name: "A", value: 1, time: now.Add(-time.Hour)}, nil)
	store.update(metric{mtype: counter, name: "B", value: 1, time: now.Add(-time.Minute)}, nil)
	store.update(metric{mtype: counter, name: "C", value: 1, time: now.Add(-time.Second)}, nil)
	store.update(metric{mtype: counter, name: "D", value: 1, time: now}, nil)
	store.update(metric{mtype: counter, name: "E", value: 1, time: now.Add(time.Second)}, nil)

	wg := sync.WaitGroup{}
	wg.Add(8)

	cleanup := func(exp time.Time) {
		store.cleanup(exp)
		wg.Done()
	}

	// The race detector should complain if there's something wrong about the
	// synchronization mechanism in the store.
	go cleanup(now.Add(-time.Hour))
	go cleanup(now.Add(-time.Hour))

	go cleanup(now.Add(-time.Minute))
	go cleanup(now.Add(-time.Minute))

	go cleanup(now.Add(-time.Second))
	go cleanup(now.Add(-time.Second))

	go cleanup(now)
	go cleanup(now)

	wg.Wait()

	metrics := store.collect(nil)
	sort.Sort(byNameAndLabels(metrics))

	if !reflect.DeepEqual(metrics, []metric{
		{mtype: counter, name: "E", value: 1, time: now.Add(time.Second), labels: labels{}},
	}) {
		t.Errorf("bad metrics: %#v", metrics)
	}
}

func BenchmarkLE(b *testing.B) {
	buckets := []stats.Value{
		stats.ValueOf(0.001),
		stats.ValueOf(0.01),
		stats.ValueOf(0.1),
		stats.ValueOf(1),
		stats.ValueOf(1),
		stats.ValueOf(math.Inf(+1)),
	}

	for i := 0; i != b.N; i++ {
		le(buckets)
	}
}
