package prometheus

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/segmentio/stats/v5"
)

type metricType int

const (
	untyped metricType = iota
	counter
	gauge
	histogram
	summary
)

func (t metricType) String() string {
	switch t {
	case untyped:
		return "untyped"
	case counter:
		return "counter"
	case gauge:
		return "gauge"
	case histogram:
		return "histogram"
	case summary:
		return "summary"
	default:
		return "unknown"
	}
}

type metricKey struct {
	scope string
	name  string
}

type metric struct {
	mtype  metricType
	scope  string
	name   string
	help   string
	value  float64
	time   time.Time
	labels labels
}

func (m metric) key() metricKey {
	return metricKey{scope: m.scope, name: m.name}
}

func (m metric) rootName() string {
	if m.mtype == histogram {
		return m.name[:strings.LastIndexByte(m.name, '_')]
	}
	return m.name
}

type metricStore struct {
	mutex   sync.RWMutex
	entries map[metricKey]*metricEntry

	// Every name the store has ever held, which cleanup deliberately does not
	// prune: exposedName has to keep deciding the same way after a colliding
	// entry expires, or a counter would change the name it publishes under
	// mid-process. Metric names come from the program rather than from the
	// data, so this is bounded by its vocabulary — label cardinality lives in
	// metricEntry.states, not here.
	names map[metricKey]struct{}
}

func (store *metricStore) lookup(mtype metricType, key metricKey, help string) *metricEntry {
	store.mutex.RLock()
	entry := store.entries[key]
	store.mutex.RUnlock()

	// The program may choose to change the type of a metric, this is likely a
	// pretty bad idea but I don't think we have enough context here to tell if
	// it's a bug or a feature so we just accept to mutate the entry.
	if entry == nil || entry.mtype != mtype {
		store.mutex.Lock()

		if store.entries == nil {
			store.entries = make(map[metricKey]*metricEntry)
		}

		if entry = store.entries[key]; entry == nil || entry.mtype != mtype {
			entry = newMetricEntry(mtype, key.scope, key.name, help)
			store.entries[key] = entry

			if store.names == nil {
				store.names = make(map[metricKey]struct{})
			}
			store.names[key] = struct{}{}
		}

		store.mutex.Unlock()
	}

	return entry
}

func (store *metricStore) update(metric metric, buckets []stats.Value) {
	entry := store.lookup(metric.mtype, metric.key(), metric.help)
	state := entry.lookup(metric.labels)
	state.update(metric.mtype, metric.value, metric.time, buckets)
}

func (store *metricStore) collect(metrics []metric) []metric {
	store.mutex.RLock()

	for key, entry := range store.entries {
		metrics = entry.collect(metrics, store.exposedName(key, entry))
	}

	store.mutex.RUnlock()
	return metrics
}

// exposedName returns the name entry publishes under.
//
// It is entry.name except where the _total suffix added to a counter would
// land on a name some other field in the same scope already occupies: a
// counter "hits" and a sibling field "hits_total" both render
// <scope>_hits_total, one # TYPE line covers the pair, and a scraper keeps
// one of the two samples. Dropping the suffix costs the counter a naming
// convention; keeping it costs a series.
//
// The answer depends on which names the store has seen, never on the order
// they arrived in or on which are live right now. Consulting the live entries
// instead would let a counter switch names once its colliding sibling expired
// — a rename mid-process, which a scraper reads as one series going stale and
// another appearing.
//
// Callers hold store.mutex.
func (store *metricStore) exposedName(key metricKey, entry *metricEntry) string {
	if entry.mtype != counter || entry.name == key.name {
		return entry.name
	}
	if _, taken := store.names[metricKey{scope: key.scope, name: entry.name}]; taken {
		return key.name
	}
	return entry.name
}

func (store *metricStore) cleanup(exp time.Time) {
	store.mutex.RLock()

	for name, entry := range store.entries {
		store.mutex.RUnlock()

		entry.cleanup(exp, func() {
			store.mutex.Lock()
			delete(store.entries, name)
			store.mutex.Unlock()
		})

		store.mutex.RLock()
	}

	store.mutex.RUnlock()
}

type metricEntry struct {
	mutex  sync.RWMutex
	mtype  metricType
	scope  string
	name   string
	help   string
	bucket string
	sum    string
	count  string
	states metricStateMap
}

func newMetricEntry(mtype metricType, scope, name, help string) *metricEntry {
	entry := &metricEntry{
		mtype:  mtype,
		scope:  scope,
		name:   name,
		help:   help,
		states: make(metricStateMap),
	}

	// Here we cache those metric names to avoid having to recompute them
	// every time we collect the state of the metrics.
	switch mtype {
	case counter:
		// Prometheus expects an accumulating count to carry a "total" suffix.
		//
		// A name that already ends in _total is left alone, so a program that
		// has already adopted the convention does not end up with
		// requests_total_total.
		if !hasTotalSuffix(name) {
			entry.name = name + "_total"
		}

	case histogram:
		entry.bucket = name + "_bucket"
		entry.sum = name + "_sum"
		entry.count = name + "_count"
	}

	return entry
}

// hasTotalSuffix reports whether name will already end in _total once it is
// exposed.
//
// The test has to run on the rendered name rather than the one received. A
// field is joined to its scope by an "_", so Incr("requests.total") arrives
// here as the bare field "total" and is published as <scope>_requests_total —
// already suffixed. Any byte invalid in a metric name, "." among them, also
// becomes "_" on the way out, so "requests.total" renders as requests_total.
// Comparing against the raw name misses both and yields _total_total.
func hasTotalSuffix(name string) bool {
	b := appendMetricName(make([]byte, 0, len(name)), name)
	return string(b) == "total" || strings.HasSuffix(string(b), "_total")
}

func (entry *metricEntry) lookup(labels labels) *metricState {
	key := labels.hash()

	entry.mutex.RLock()
	state := entry.states.find(key, labels)
	entry.mutex.RUnlock()

	if state == nil {
		entry.mutex.Lock()

		if state = entry.states.find(key, labels); state == nil {
			state = newMetricState(labels)
			entry.states.put(key, state)
		}

		entry.mutex.Unlock()
	}

	return state
}

func (entry *metricEntry) collect(metrics []metric, name string) []metric {
	entry.mutex.RLock()

	if len(entry.states) != 0 {
		for _, states := range entry.states {
			for _, state := range states {
				metrics = state.collect(metrics, entry, name)
			}
		}
	}

	entry.mutex.RUnlock()
	return metrics
}

func (entry *metricEntry) cleanup(exp time.Time, empty func()) {
	// TODO: there may be high contention on this mutex, maybe not, it would be
	// a good idea to measure.
	entry.mutex.Lock()

	for hash, states := range entry.states {
		i := 0

		for j, state := range states {
			states[j] = nil
			state.mutex.Lock()

			// We expire all entries that have been last updated before exp,
			// they don't get copied back into the state slice.
			if exp.Before(state.time) {
				states[i] = state
				i++
			}

			state.mutex.Unlock()
		}

		if states = states[:i]; len(states) == 0 {
			delete(entry.states, hash)
		} else {
			entry.states[hash] = states
		}
	}

	// remember whether we emptied out all states _while_ holding the lock
	shouldDelete := (len(entry.states) == 0)
	entry.mutex.Unlock()

	// now call back into store (taking store.mutex) only after releasing entry.mutex
	if shouldDelete {
		empty()
	}
}

type metricState struct {
	// immutable
	labels labels
	// mutable
	mutex   sync.Mutex
	buckets metricBuckets
	value   float64
	sum     float64
	count   uint64
	time    time.Time
}

func newMetricState(labels labels) *metricState {
	return &metricState{
		labels: labels.copy(),
	}
}

func (state *metricState) update(mtype metricType, value float64, time time.Time, buckets []stats.Value) {
	state.mutex.Lock()

	switch mtype {
	case counter:
		state.value += value

	case gauge:
		state.value = value

	case histogram:
		// makeMetricBuckets appends a +Inf bucket, so the state holds one more
		// entry than the registry slice. Comparing against len(buckets) here
		// would rebuild — and zero the counts — on every observation.
		if len(state.buckets) != len(buckets)+1 {
			state.buckets = makeMetricBuckets(buckets, state.labels)
		}
		state.buckets.update(value)
		state.sum += value
		state.count++
	}

	state.time = time
	state.mutex.Unlock()
}

func (state *metricState) collect(metrics []metric, entry *metricEntry, name string) []metric {
	state.mutex.Lock()

	// metric.time is deliberately not set here. appendMetric no longer writes
	// a timestamp, so nothing on the output path reads it; the field stays on
	// the struct for the input path, where metricStore.update carries it into
	// state.time and MetricTimeout expiry depends on it.
	switch entry.mtype {
	case counter, gauge:
		metrics = append(metrics, metric{
			mtype:  entry.mtype,
			scope:  entry.scope,
			name:   name,
			help:   entry.help,
			value:  state.value,
			labels: state.labels,
		})

	case histogram:
		// Prometheus' scraper expects for histogram buckets to be cumulative.
		// [1] https://prometheus.io/docs/practices/histograms/#apdex-score
		// [2] https://en.wikipedia.org/wiki/Histogram#Cumulative_histogram
		var cumulativeCount uint64
		for _, bucket := range state.buckets {
			cumulativeCount += bucket.count
			metrics = append(metrics, metric{
				mtype:  entry.mtype,
				scope:  entry.scope,
				name:   entry.bucket,
				help:   entry.help,
				value:  float64(cumulativeCount),
				labels: bucket.labels,
			})
		}
		metrics = append(metrics,
			metric{
				mtype:  entry.mtype,
				scope:  entry.scope,
				name:   entry.sum,
				help:   entry.help,
				value:  state.sum,
				labels: state.labels,
			},
			metric{
				mtype:  entry.mtype,
				scope:  entry.scope,
				name:   entry.count,
				help:   entry.help,
				value:  float64(state.count),
				labels: state.labels,
			},
		)
	}

	state.mutex.Unlock()
	return metrics
}

type metricStateMap map[uint64][]*metricState

func (m metricStateMap) put(key uint64, state *metricState) {
	m[key] = append(m[key], state)
}

func (m metricStateMap) find(key uint64, labels labels) *metricState {
	states := m[key]

	for _, state := range states {
		if state.labels.equal(labels) {
			return state
		}
	}

	return nil
}

type metricBucket struct {
	limit  float64
	count  uint64
	labels labels
}

type metricBuckets []metricBucket

// makeMetricBuckets builds the bucket set for a histogram state, with one
// entry per registered boundary plus a final +Inf bucket.
//
// The +Inf bucket is not optional: histogram_quantile returns NaN unless the
// highest bucket has an upper bound of +Inf, and without it observations above
// the last registered boundary are counted in _sum and _count but land in no
// bucket at all.
//
// Callers that compare an existing bucket set against the registry slice to
// decide whether to rebuild must account for the extra entry.
func makeMetricBuckets(buckets []stats.Value, labels labels) metricBuckets {
	b := make(metricBuckets, len(buckets)+1)
	s := le(buckets)

	for i := range buckets {
		var le string
		le, s = nextLe(s)
		b[i].limit = valueOf(buckets[i])
		b[i].labels = labels.copyAppend(label{"le", le})
	}

	b[len(buckets)].limit = math.Inf(1)
	b[len(buckets)].labels = labels.copyAppend(label{"le", "+Inf"})

	return b
}

func (m metricBuckets) update(value float64) {
	for i := range m {
		if value <= m[i].limit {
			m[i].count++
			break
		}
	}
}

// This function builds a string of column-separated float representations of
// the given list of buckets, which is then split by calls to nextLe to generate
// the values of the "le" label for each bucket of a histogram.
//
// The intent is to keep the number of dynamic memory allocations constant
// instead of increasing linearly with the number of buckets.
func le(buckets []stats.Value) string {
	if len(buckets) == 0 {
		return ""
	}

	b := make([]byte, 0, 8*len(buckets))

	for i, v := range buckets {
		if i != 0 {
			b = append(b, ':')
		}
		b = appendFloat(b, valueOf(v))
	}
	return unsafeByteSliceToString(b)
}

// unsafeByteSliceToString converts a byte slice to a string without copying the underlying data.
func unsafeByteSliceToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func nextLe(s string) (head, tail string) {
	if before, after, ok := strings.Cut(s, ":"); ok {
		head, tail = before, after
	} else {
		head = s
	}
	return head, tail
}

func appendFloat(b []byte, f float64) []byte {
	return strconv.AppendFloat(b, f, 'g', -1, 64)
}

type byNameAndLabels []metric

func (metrics byNameAndLabels) Len() int {
	return len(metrics)
}

func (metrics byNameAndLabels) Swap(i, j int) {
	metrics[i], metrics[j] = metrics[j], metrics[i]
}

// Less orders by scope before name, so that every metric sharing a scope and
// a root name stays contiguous in the output.
//
// Ordering on the bare name would interleave same-named fields coming from
// different engine prefixes — hits from two WithPrefix sub-engines, say —
// which breaks up the family a single "# TYPE" line is meant to cover.
//
// Comparing the two parts in turn rather than the joined "scope_name" avoids
// building a string for every comparison in the sort.
func (metrics byNameAndLabels) Less(i, j int) bool {
	m1 := &metrics[i]
	m2 := &metrics[j]

	if m1.scope != m2.scope {
		return m1.scope < m2.scope
	}
	if m1.name != m2.name {
		return m1.name < m2.name
	}
	return m1.labels.less(m2.labels)
}
