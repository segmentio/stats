package prometheus

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/segmentio/stats"
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

	for _, entry := range store.entries {
		metrics = entry.collect(metrics)
	}

	store.mutex.RUnlock()
	return metrics
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

func newMetricEntry(mtype metricType, scope string, name string, help string) *metricEntry {
	entry := &metricEntry{
		mtype:  mtype,
		scope:  scope,
		name:   name,
		help:   help,
		states: make(metricStateMap),
	}

	if mtype == histogram {
		// Here we cache those metric names to avoid having to recompute them
		// every time we collect the state of the metrics.
		entry.bucket = name + "_bucket"
		entry.sum = name + "_sum"
		entry.count = name + "_count"
	}

	return entry
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

func (entry *metricEntry) collect(metrics []metric) []metric {
	entry.mutex.RLock()

	if len(entry.states) != 0 {
		for _, states := range entry.states {
			for _, state := range states {
				metrics = state.collect(metrics, entry)
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

	if len(entry.states) == 0 {
		empty()
	}

	entry.mutex.Unlock()
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
		if len(state.buckets) != len(buckets) {
			state.buckets = makeMetricBuckets(buckets, state.labels)
		}
		state.buckets.update(value)
		state.sum += value
		state.count++
	}

	state.time = time
	state.mutex.Unlock()
}

func (state *metricState) collect(metrics []metric, entry *metricEntry) []metric {
	state.mutex.Lock()

	switch entry.mtype {
	case counter, gauge:
		metrics = append(metrics, metric{
			mtype:  entry.mtype,
			scope:  entry.scope,
			name:   entry.name,
			help:   entry.help,
			value:  state.value,
			time:   state.time,
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
				time:   state.time,
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
				time:   state.time,
				labels: state.labels,
			},
			metric{
				mtype:  entry.mtype,
				scope:  entry.scope,
				name:   entry.count,
				help:   entry.help,
				value:  float64(state.count),
				time:   state.time,
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

func makeMetricBuckets(buckets []stats.Value, labels labels) metricBuckets {
	b := make(metricBuckets, len(buckets))
	s := le(buckets)

	for i := range buckets {
		var le string
		le, s = nextLe(s)
		b[i].limit = valueOf(buckets[i])
		b[i].labels = labels.copyAppend(label{"le", le})
	}

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

	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		Data: uintptr(unsafe.Pointer(&b[0])),
		Len:  len(b),
	}))
}

func nextLe(s string) (head string, tail string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		head, tail = s[:i], s[i+1:]
	} else {
		head = s
	}
	return
}

func appendFloat(b []byte, f float64) []byte {
	return strconv.AppendFloat(b, f, 'g', -1, 64)
}

type byNameAndLabels []metric

func (metrics byNameAndLabels) Len() int {
	return len(metrics)
}

func (metrics byNameAndLabels) Swap(i int, j int) {
	metrics[i], metrics[j] = metrics[j], metrics[i]
}

func (metrics byNameAndLabels) Less(i int, j int) bool {
	m1 := &metrics[i]
	m2 := &metrics[j]
	return m1.name < m2.name || (m1.name == m2.name && m1.labels.less(m2.labels))
}
