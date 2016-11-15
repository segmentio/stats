package stats

import (
	"sort"
	"time"
)

type MetricType int

const (
	CounterType MetricType = iota
)

type Metric struct {
	Type  MetricType
	Key   string
	Name  string
	Tags  []Tag
	Value float64
}

type NaturalMetricOrder []Metric

func (m NaturalMetricOrder) Less(i int, j int) bool {
	return m[i].Key < m[j].Key
}

func (m NaturalMetricOrder) Swap(i int, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m NaturalMetricOrder) Len() int {
	return len(m)
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

type metricState struct {
	typ     MetricType
	name    string
	tags    []Tag
	value   float64
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
		})
	}

	sort.Sort(NaturalMetricOrder(metrics))
	return metrics
}

func (s metricStore) update(m Metric, now time.Time) {
	state := s.metrics[m.Key]

	if state == nil {
		state = &metricState{
			typ:  m.Type,
			name: m.Name,
			tags: m.Tags,
		}
		s.metrics[m.Key] = state
	}

	switch m.Type {
	case CounterType:
		state.value += m.Value
	}

	state.expTime = now.Add(s.timeout)
}

func (s metricStore) deleteExpiredMetrics(now time.Time) {
	for key, state := range s.metrics {
		if now.After(state.expTime) {
			delete(s.metrics, key)
		}
	}
}
