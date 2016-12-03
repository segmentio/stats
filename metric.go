package stats

import "time"

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

// The Namespace type represents the namespace in which a metric exists.
type Namespace struct {
	Name string // The name of the namespace.
	Tags []Tag  // The tags to apply to all metrics of the namespace.
}

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
	// assume that it will be uniquely representing a single metric within an
	// engine.
	Key string

	// Name is the name of the metric as defined by the program.
	Name string

	// Tags is the list of tags set on the metric.
	Tags []Tag

	// Value is the current value of the metric.
	Value float64

	// Time is set to the time at which the metric was last modified.
	Time time.Time

	// Namespace carries the metric namespace.
	Namespace Namespace
}

// MetricKey takes the name and tags of a metric and returns a unique key
// representing that metric.
func MetricKey(name string, tags []Tag) string {
	return string(appendMetricKey(make([]byte, 0, metricKeyLen(name, tags)), name, tags))
}

// RawMetricKey works the same as MetricKey but receives the tags as a RawTags
// object.
func RawMetricKey(name string, tags RawTags) string {
	return name + "?" + string(tags)
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

// The MetricsByKey type implements sort.Interface and can be used to sort a
// slice of metrics by key.
type MetricsByKey []Metric

// Less returns true if the metric key at index i is ordered before the metric
// key at index j.
func (m MetricsByKey) Less(i int, j int) bool {
	return m[i].Key < m[j].Key
}

// Swap swaps the metrics at index i and j.
func (m MetricsByKey) Swap(i int, j int) {
	m[i], m[j] = m[j], m[i]
}

// Len returns the lengths of the metric slice.
func (m MetricsByKey) Len() int {
	return len(m)
}

type metricOp struct {
	typ   MetricType
	space Namespace
	key   string
	name  string
	tags  []Tag
	value float64
	time  time.Time
	apply func(*metricState, float64, time.Time, time.Time, uint64)
}

func metricOpAdd(state *metricState, value float64, mod time.Time, exp time.Time, version uint64) {
	state.value += value
	state.version = version
	state.modTime = mod
	state.expTime = exp
}

func metricOpSet(state *metricState, value float64, mod time.Time, exp time.Time, version uint64) {
	state.value = value
	state.version = version
	state.modTime = mod
	state.expTime = exp
}

func metricOpObserve(state *metricState, value float64, mod time.Time, exp time.Time, version uint64) {
	if state.metrics == nil {
		state.metrics = make([]metricSubState, 0, 100)
	}
	state.version = version
	state.modTime = mod
	state.expTime = exp
	state.metrics = append(state.metrics, metricSubState{
		value:   value,
		version: version,
		modTime: mod,
		expTime: exp,
	})
}

type metricReq struct {
	res   chan<- metricRes
	since uint64
}

type metricRes struct {
	metrics []Metric
	version uint64
}

type metricState struct {
	typ     MetricType
	space   Namespace
	key     string
	name    string
	tags    []Tag
	value   float64
	version uint64
	modTime time.Time
	expTime time.Time
	metrics []metricSubState // observed values
}

type metricSubState struct {
	value   float64
	version uint64
	modTime time.Time
	expTime time.Time
}

type metricStore struct {
	metrics map[string]*metricState
	timeout time.Duration
	version uint64
}

type metricStoreConfig struct {
	timeout time.Duration
}

func newMetricStore(config metricStoreConfig) *metricStore {
	return &metricStore{
		metrics: make(map[string]*metricState),
		timeout: config.timeout,
	}
}

func (s *metricStore) state(lastVersion uint64) (metrics []Metric, version uint64) {
	metrics = make([]Metric, 0, 2*len(s.metrics))

	for _, state := range s.metrics {
		if state.version <= lastVersion {
			continue
		}

		if len(state.metrics) == 0 {
			metrics = append(metrics, Metric{
				Type:      state.typ,
				Key:       state.key,
				Name:      state.name,
				Tags:      state.tags,
				Value:     state.value,
				Time:      state.modTime,
				Namespace: state.space,
			})
			continue
		}

		for _, sub := range state.metrics {
			if sub.version > lastVersion {
				metrics = append(metrics, Metric{
					Type:      state.typ,
					Key:       state.key,
					Name:      state.name,
					Tags:      state.tags,
					Value:     sub.value,
					Time:      sub.modTime,
					Namespace: state.space,
				})
			}
		}
	}

	version = s.version
	return
}

func (s *metricStore) apply(op metricOp, now time.Time) {
	state := s.metrics[op.key]

	if state == nil || state.typ != op.typ {
		state = &metricState{
			typ:   op.typ,
			space: op.space,
			key:   op.key,
			name:  op.name,
			tags:  op.tags,
		}
		s.metrics[op.key] = state
	}

	if op.time == (time.Time{}) {
		op.time = now
	}

	s.version++
	op.apply(state, op.value, op.time, now.Add(s.timeout), s.version)
}

func (s *metricStore) deleteExpiredMetrics(now time.Time) {
	for key, state := range s.metrics {
		if now.After(state.expTime) {
			delete(s.metrics, key)
			continue
		}

		i := 0

		for _, sub := range state.metrics {
			if !now.After(sub.expTime) {
				state.metrics[i] = sub
				i++
			}
		}

		state.metrics = state.metrics[:i]
	}
}
