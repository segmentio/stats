package stats

import (
	"sort"
	"time"
)

// MetricType is an enumeration representing the type of a metric.
type MetricType int

const (
	// CounterType is the constant representing counters.
	CounterType MetricType = iota

	// GaugeType is the constant representing gauges.
	GaugeType
)

// Metric is a universal representation of the state of a metric.
//
// No operations are available on this data type, instead it carries the state
// of a metric a single metric when querying the state of a stats engine.
type Metric struct {
	// Type is a constant representing the type of the metric, which is one of
	// the constants defined by the MetricType enumeration.
	Type MetricType

	// Key is a unique identifier for the metric.
	//
	// Application should not rely on the actual structure of the key and just
	// assume that it will be uniquely representing a single metric.
	Key string

	// Name is the name of the metric as defined by the program.
	Name string

	// Tags is the list of tags set on the metric.
	Tags []Tag

	// Value is the current value of a metric.
	//
	// This field is only valid for counters and gauge.
	Value float64

	// Count is a counter of the number of operations that have been done on a
	// metric.
	//
	// Note that for a single metric this value may not always increase. If a
	// metric is idle for too long and times out, then is produced again later,
	// the count will be set back to one.
	Count uint64
}

type metricsByKey []Metric

func (m metricsByKey) Less(i int, j int) bool {
	return m[i].Key < m[j].Key
}

func (m metricsByKey) Swap(i int, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m metricsByKey) Len() int {
	return len(m)
}

func sortMetrics(metrics []Metric) {
	sort.Sort(metricsByKey(metrics))
}

func metricKey(name string, tags []Tag) string {
	return string(appendMetricKey(make([]byte, 0, metricKeyLen(name, tags)), name, tags))
}

func metricKeyLen(name string, tags []Tag) int {
	return len(name) + 1 + tagsLen(tags)
}

func appendMetricKey(b []byte, name string, tags []Tag) []byte {
	b = append(b, name...)
	b = append(b, '?')
	b = appendTags(b, tags)
	return b
}

type metricOp struct {
	typ   MetricType
	key   string
	name  string
	tags  []Tag
	value float64
	apply func(*metricState, float64)
}

func metricOpAdd(state *metricState, value float64) {
	state.value += value
}

func metricOpSub(state *metricState, value float64) {
	state.value -= value
}

func metricOpSet(state *metricState, value float64) {
	state.value = value
}

type metricReq struct {
	res chan<- []Metric
}

type metricState struct {
	typ     MetricType
	name    string
	tags    []Tag
	value   float64
	count   uint64
	expTime time.Time
}

type metricStore struct {
	metrics map[string]*metricState
	timeout time.Duration
}

type metricStoreConfig struct {
	timeout time.Duration
}

func makeMetricStore(config metricStoreConfig) metricStore {
	return metricStore{
		metrics: make(map[string]*metricState),
		timeout: config.timeout,
	}
}

func (s metricStore) state() []Metric {
	metrics := make([]Metric, 0, len(s.metrics))

	for key, state := range s.metrics {
		metrics = append(metrics, Metric{
			Key:   key,
			Type:  state.typ,
			Name:  state.name,
			Tags:  state.tags,
			Value: state.value,
			Count: state.count,
		})
	}

	return metrics
}

func (s metricStore) apply(op metricOp, now time.Time) {
	state := s.metrics[op.key]

	if state == nil {
		state = &metricState{
			typ:  op.typ,
			name: op.name,
			tags: op.tags,
		}
		s.metrics[op.key] = state
	}

	op.apply(state, op.value)
	state.count++
	state.expTime = now.Add(s.timeout)
}

func (s metricStore) deleteExpiredMetrics(now time.Time) {
	for key, state := range s.metrics {
		if now.After(state.expTime) {
			delete(s.metrics, key)
		}
	}
}
