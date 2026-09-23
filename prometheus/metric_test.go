package prometheus

import (
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
)

func TestFuzzyDeadlock(t *testing.T) {
	const (
		iterations = 10000
		timeout    = 1 * time.Second
	)

	for i := range iterations {
		// 1) fresh store with one expired metric so cleanup() will actually delete
		var store metricStore
		store.entries = make(map[metricKey]*metricEntry)
		m := metric{
			mtype: counter,
			scope: "svc",
			name:  "fuzzy_deadlock",
			value: 1,
			time:  time.Now().Add(-time.Hour), // expired
		}
		store.update(m, nil)

		// 2) race collect vs cleanup once
		done := make(chan struct{}, 2)
		go func() {
			store.collect(nil)
			done <- struct{}{}
		}()
		go func() {
			store.cleanup(time.Now())
			done <- struct{}{}
		}()

		// 3) both must finish within timeout or we assume a deadlock
		for range 2 {
			select {
			case <-done:
				// one of them completed
			case <-time.After(timeout):
				t.Fatalf("iteration %d: deadlock (neither collect nor cleanup returned)", i)
			}
		}
	}
}

func TestUnsafeByteSliceToString(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "nil bytes",
			input:    nil,
			expected: "",
		},
		{
			name:     "no bytes",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "list of floats",
			input:    []byte("1.2:3.4:5.6:7.8"),
			expected: "1.2:3.4:5.6:7.8",
		},
		{
			name:     "deadbeef",
			input:    []byte{0xde, 0xad, 0xbe, 0xef},
			expected: "\xde\xad\xbe\xef",
		},
		{
			name:     "embedded zero",
			input:    []byte("this\x00that"),
			expected: "this\x00that",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := unsafeByteSliceToString(test.input)
			if res != test.expected {
				t.Errorf("Expected %q but got %q", test.expected, res)
			}
		})
	}
}

func TestMetricStore(t *testing.T) {
	input := []metric{
		{mtype: counter, scope: "test", name: "A", value: 1},
		{mtype: counter, scope: "test", name: "A", value: 2},
		{mtype: histogram, scope: "test", name: "C", value: 0.1},
		{mtype: gauge, scope: "test", name: "B", value: 1, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: counter, scope: "test", name: "A_total", value: 4, labels: labels{{"id", "123"}}},
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
		// "A" would be suffixed to A_total, which the sibling field A_total
		// already occupies — one # TYPE line over two samples, of which a
		// scraper keeps one. The suffix is dropped instead, leaving two
		// distinct families.
		{mtype: counter, scope: "test", name: "A", value: 3, labels: labels{}},
		{mtype: counter, scope: "test", name: "A_total", value: 4, labels: labels{{"id", "123"}}},
		{mtype: gauge, scope: "test", name: "B", value: 42, labels: labels{{"a", "1"}}},
		{mtype: gauge, scope: "test", name: "B", value: 21, labels: labels{{"a", "1"}, {"b", "2"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 2, labels: labels{{"le", "0.25"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "0.5"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "0.75"}}},
		{mtype: histogram, scope: "test", name: "C_bucket", value: 3, labels: labels{{"le", "1"}}},
		// The 10 observation exceeds the highest registered boundary, so the
		// +Inf bucket matches C_count. It sorts last because le compares
		// numerically.
		{mtype: histogram, scope: "test", name: "C_bucket", value: 4, labels: labels{{"le", "+Inf"}}},
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
				{value: 42, time: now},
				{value: 1, time: now.Add(-time.Minute)},
				{value: 2, time: now.Add(-(500 * time.Millisecond))},
			},
			1: []*metricState{
				{value: 123, time: now.Add(10 * time.Millisecond)},
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
			{value: 42, time: now},
			{value: 2, time: now.Add(-(500 * time.Millisecond))},
		},
		1: []*metricState{
			{value: 123, time: now.Add(10 * time.Millisecond)},
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
			{value: 123, time: now.Add(10 * time.Millisecond)},
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

	// collect() does not carry state.time onto the collected metric: nothing
	// reads it since appendMetric stopped emitting timestamps. The input times
	// above are what drive expiry, and which entries survive is what this
	// asserts.
	if !reflect.DeepEqual(metrics, []metric{
		{mtype: counter, name: "E_total", value: 1, labels: labels{}},
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

	for b.Loop() {
		le(buckets)
	}
}

// TestMetricStateBucketsNotRebuilt guards the interaction between the +Inf
// bucket and the rebuild check in metricState.update.
//
// makeMetricBuckets returns len(buckets)+1 entries. If update compares the
// stored slice against len(buckets) it never matches, so every observation
// reallocates the bucket set and discards the counts accumulated so far —
// leaving _count climbing while every _bucket stays at 0 or 1.
func TestMetricStateBucketsNotRebuilt(t *testing.T) {
	buckets := []stats.Value{
		stats.ValueOf(0.25),
		stats.ValueOf(0.5),
	}

	state := newMetricState(labels{})

	const observations = 10
	for range observations {
		state.update(histogram, 0.1, time.Now(), buckets)
	}

	if n := len(state.buckets); n != len(buckets)+1 {
		t.Fatalf("expected %d buckets (registered + Inf), found %d", len(buckets)+1, n)
	}

	// Every observation is 0.1, so all of them belong in the lowest bucket.
	if c := state.buckets[0].count; c != observations {
		t.Errorf("buckets were rebuilt: le=0.25 has count %d, expected %d", c, observations)
	}

	if state.count != observations {
		t.Errorf("count = %d, expected %d", state.count, observations)
	}
}

// TestMakeMetricBucketsAppendsInf covers the empty-registry case, where the
// +Inf bucket is the only one, and confirms it catches overflow.
func TestMakeMetricBucketsAppendsInf(t *testing.T) {
	for _, test := range []struct {
		name    string
		buckets []stats.Value
	}{
		{name: "empty", buckets: nil},
		{name: "one boundary", buckets: []stats.Value{stats.ValueOf(1.0)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := makeMetricBuckets(test.buckets, labels{})

			if len(b) != len(test.buckets)+1 {
				t.Fatalf("expected %d buckets, found %d", len(test.buckets)+1, len(b))
			}

			last := b[len(b)-1]
			if !math.IsInf(last.limit, 1) {
				t.Errorf("last bucket limit = %v, expected +Inf", last.limit)
			}
			if got := last.labels[len(last.labels)-1]; got != (label{"le", "+Inf"}) {
				t.Errorf("last bucket label = %v, expected le=+Inf", got)
			}

			// A value above every registered boundary must still be counted.
			b.update(1e9)
			if c := b[len(b)-1].count; c != 1 {
				t.Errorf("overflow observation not counted in +Inf bucket (count = %d)", c)
			}
		})
	}
}

// TestCounterTotalSuffixUsesRenderedName pins the suffix check to the name as exposed rather
// than as received. A field is joined to its scope by an "_", so Incr("x.total")
// arrives here as the bare field "total" and is already suffixed once written;
// "." also renders as "_", so a dotted name can be suffixed too. Testing the
// raw name misses both and publishes _total_total.
func TestCounterTotalSuffixUsesRenderedName(t *testing.T) {
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "hits", want: "hits_total"},
		{name: "hits_total", want: "hits_total"},
		{name: "total", want: "total"},
		{name: "requests.total", want: "requests.total"},
		{name: "subtotal", want: "subtotal_total"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if entry := newMetricEntry(counter, "app", test.name, ""); entry.name != test.want {
				t.Errorf("newMetricEntry(%q).name = %q, expected %q",
					test.name, entry.name, test.want)
			}
		})
	}
}

// TestCounterTotalSuffixDottedName is the same invariant at the other end:
// e.Incr("requests.total") must not publish app_requests_total_total.
func TestCounterTotalSuffixDottedName(t *testing.T) {
	h := &Handler{}

	// What Incr("requests.total") produces: measure "app.requests", field "total".
	h.HandleMeasures(time.Now(), stats.Measure{
		Name:   "app.requests",
		Fields: []stats.Field{stats.MakeField("total", 1, stats.Counter)},
	})

	var buf strings.Builder
	h.WriteStats(&buf)
	out := buf.String()

	if !strings.Contains(out, "app_requests_total 1") {
		t.Errorf("expected app_requests_total:\n%s", out)
	}
	if strings.Contains(out, "total_total") {
		t.Errorf("counter was double-suffixed:\n%s", out)
	}
}

// TestCounterTotalSuffixCollision covers the pair the suffix rule can bring
// onto one name. Both families have to survive, whichever order they arrive
// in, so that neither sample is dropped by the scraper.
func TestCounterTotalSuffixCollision(t *testing.T) {
	for _, order := range [][]string{
		{"hits", "hits_total"},
		{"hits_total", "hits"},
	} {
		t.Run(strings.Join(order, ","), func(t *testing.T) {
			h := &Handler{}

			for _, name := range order {
				h.HandleMeasures(time.Now(), stats.Measure{
					Name:   "svc",
					Fields: []stats.Field{stats.MakeField(name, 1, stats.Counter)},
				})
			}

			var buf strings.Builder
			h.WriteStats(&buf)
			out := buf.String()

			for _, want := range []string{
				"# TYPE svc_hits counter",
				"svc_hits 1",
				"# TYPE svc_hits_total counter",
				"svc_hits_total 1",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("missing %q in output:\n%s", want, out)
				}
			}

			// One sample each. Two would mean they collided onto one name.
			if n := strings.Count(out, "svc_hits_total 1"); n != 1 {
				t.Errorf("found %d svc_hits_total samples, expected 1:\n%s", n, out)
			}
		})
	}
}

// TestCounterTotalSuffixCollisionSurvivesCleanup covers the collision outliving
// the entry that caused it. The sibling stops being reported and is swept by
// the MetricTimeout cleanup; the counter must keep the name it has been
// publishing under, because changing it reads to a scraper as one series going
// stale and another appearing.
func TestCounterTotalSuffixCollisionSurvivesCleanup(t *testing.T) {
	now := time.Now()
	h := &Handler{}

	h.HandleMeasures(now, stats.Measure{
		Name:   "svc",
		Fields: []stats.Field{stats.MakeField("hits", 1, stats.Counter)},
	})
	h.HandleMeasures(now.Add(-time.Hour), stats.Measure{
		Name:   "svc",
		Fields: []stats.Field{stats.MakeField("hits_total", 5, stats.Counter)},
	})

	var buf strings.Builder
	h.WriteStats(&buf)
	if out := buf.String(); !strings.Contains(out, "svc_hits 1") {
		t.Fatalf("expected svc_hits before cleanup:\n%s", out)
	}

	h.metrics.cleanup(now.Add(-2 * time.Minute))

	buf.Reset()
	h.WriteStats(&buf)
	out := buf.String()

	if !strings.Contains(out, "svc_hits 1") {
		t.Errorf("counter renamed itself after the sibling expired:\n%s", out)
	}
	if strings.Contains(out, "svc_hits_total") {
		t.Errorf("counter took the expired sibling's name:\n%s", out)
	}
}
