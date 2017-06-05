package prometheus

import (
	"sync"
	"time"

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

func metricTypeOf(t stats.MetricType) metricType {
	switch t {
	case stats.CounterType:
		return counter
	case stats.GaugeType:
		return gauge
	case stats.HistogramType:
		return histogram
	default:
		return untyped
	}
}

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

type metric struct {
	mtype  metricType
	name   string
	help   string
	value  float64
	time   time.Time
	labels labels
}

type metricStore struct {
	mutex   sync.RWMutex
	entries map[string]*metricEntry
}

func (store *metricStore) lookup(mtype metricType, name string, help string) *metricEntry {
	store.mutex.RLock()
	entry := store.entries[name]
	store.mutex.RUnlock()

	// The program may choose to change the type of a metric, this is likely a
	// pretty bad idea but I don't think we have enough context here to tell if
	// it's a bug or a feature so we just accept to mutate the entry.
	if entry == nil || entry.mtype != mtype {
		store.mutex.Lock()

		if store.entries == nil {
			store.entries = make(map[string]*metricEntry)
		}

		if entry = store.entries[name]; entry == nil || entry.mtype != mtype {
			entry = newMetricEntry(mtype, name, help)
			store.entries[name] = entry
		}

		store.mutex.Unlock()
	}

	return entry
}

func (store *metricStore) update(metric metric) {
	entry := store.lookup(metric.mtype, metric.name, metric.help)
	state := entry.lookup(metric.labels)
	state.update(metric.mtype, metric.value, metric.time)
}

func (store *metricStore) collect(metrics []metric) []metric {
	store.mutex.RLock()

	for _, entry := range store.entries {
		metrics = entry.collect(metrics)
	}

	store.mutex.RUnlock()
	return metrics
}

type metricEntry struct {
	mutex  sync.RWMutex
	mtype  metricType
	name   string
	help   string
	states metricStateMap
}

func newMetricEntry(mtype metricType, name string, help string) *metricEntry {
	return &metricEntry{
		mtype:  mtype,
		name:   name,
		help:   help,
		states: make(metricStateMap),
	}
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
		zero := len(metrics)

		for _, states := range entry.states {
			for _, state := range states {
				metrics = state.collect(metrics)
			}
		}

		for i := range metrics[zero:] {
			m := &metrics[zero+i]
			m.mtype = entry.mtype
			m.name = entry.name
			m.help = entry.help
		}
	}

	entry.mutex.RUnlock()
	return metrics
}

type metricState struct {
	// immutable
	labels labels
	// mutable
	mutex sync.Mutex
	value float64
	time  time.Time
}

func newMetricState(labels labels) *metricState {
	return &metricState{
		labels: labels.copy(),
	}
}

func (state *metricState) update(mtype metricType, value float64, time time.Time) {
	state.mutex.Lock()

	switch mtype {
	case counter:
		state.value += value

	case gauge:
		state.value = value

	case histogram:
		// TODO
	}

	state.time = time
	state.mutex.Unlock()
}

func (state *metricState) collect(metrics []metric) []metric {
	state.mutex.Lock()
	metrics = append(metrics, metric{
		value:  state.value,
		time:   state.time,
		labels: state.labels,
	})
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
