package stats

import (
	"strconv"
	"time"
)

// MetricType is an enumeration representing the type of a metric.
type MetricType int

const (
	// CounterType is the constant representing counter metrics.
	CounterType MetricType = iota

	// GaugeType is the constant representing gauge metrics.
	GaugeType

	// HistogramType is the constant representing histogram metrics.
	HistogramType
)

// Metric is a universal representation of the state of a metric.
//
// No operations are available on this data type, instead it carries the state
// of a metric a single metric when querying the state of a stats engine.
type Metric struct {
	// Type is a constant representing the type of the metric, which is one of
	// the constants defined by the MetricType enumeration.
	Type MetricType

	// Group is a unique identifier of the group this metric belongs to.
	//
	// Not all metrics belong to groups, most of the time the group is an empty
	// string. Some metrics however are aggregates of submetrics, in that case
	// all submetrics will have the same group value which is the key of the
	// parent metric.
	Group string

	// Key is a unique identifier for the metric.
	//
	// Application should not rely on the actual structure of the key and just
	// assume that it will be uniquely representing a single metric.
	Key string

	// Name is the name of the metric as defined by the program.
	Name string

	// Tags is the list of tags set on the metric.
	Tags []Tag

	// Value is the current value of the metric.
	Value float64

	// Sample is a counter of the number of operations that have been done on a
	// metric.
	//
	// Note that for a single metric this value may not always increase. If a
	// metric is idle for too long and times out, then is produced again later,
	// the sample will be set back to one.
	Sample uint64
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
	apply func(*metricState, float64, time.Time)
}

func metricOpAdd(state *metricState, value float64, exp time.Time) {
	state.value += value
	state.sample++
	state.expTime = exp
}

func metricOpSub(state *metricState, value float64, exp time.Time) {
	state.value -= value
	state.sample++
	state.expTime = exp
}

func metricOpSet(state *metricState, value float64, exp time.Time) {
	state.value = value
	state.sample++
	state.expTime = exp
}

func metricOpObserve(state *metricState, value float64, exp time.Time) {
	key := state.key + "#" + strconv.FormatUint(state.sample, 10)
	state.sample++
	state.expTime = exp
	state.metrics[key] = metricState{
		typ:     state.typ,
		group:   state.key,
		key:     key,
		name:    state.name,
		tags:    state.tags,
		value:   value,
		sample:  1,
		expTime: exp,
	}
}

type metricReq struct {
	res chan<- []Metric
}

type metricState struct {
	typ     MetricType
	group   string
	key     string
	name    string
	tags    []Tag
	value   float64
	sample  uint64
	expTime time.Time
	metrics map[string]metricState // observed values
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
	metrics := make([]Metric, 0, 2*len(s.metrics))

	for _, state := range s.metrics {
		if len(state.metrics) == 0 {
			metrics = append(metrics, Metric{
				Type:   state.typ,
				Group:  state.group,
				Key:    state.key,
				Name:   state.name,
				Tags:   state.tags,
				Value:  state.value,
				Sample: state.sample,
			})
			continue
		}

		for _, sub := range state.metrics {
			metrics = append(metrics, Metric{
				Type:   sub.typ,
				Group:  sub.group,
				Key:    sub.key,
				Name:   sub.name,
				Tags:   sub.tags,
				Value:  sub.value,
				Sample: sub.sample,
			})
		}
	}

	return metrics
}

func (s metricStore) apply(op metricOp, now time.Time) {
	state := s.metrics[op.key]

	if state == nil || state.typ != op.typ {
		state = &metricState{
			typ:  op.typ,
			key:  op.key,
			name: op.name,
			tags: op.tags,
		}
		s.metrics[op.key] = state
	}

	op.apply(state, op.value, now.Add(s.timeout))
}

func (s metricStore) deleteExpiredMetrics(now time.Time) {
	for key, state := range s.metrics {
		if now.After(state.expTime) {
			delete(s.metrics, key)
			continue
		}

		for key, sub := range state.metrics {
			if now.After(sub.expTime) {
				delete(state.metrics, key)
			}
		}
	}
}
