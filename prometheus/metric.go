package prometheus

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

type Metric struct {
	Name   string
	Type   string
	Help   string
	Value  float64
	Time   time.Time
	Labels Labels

	// The unexported fields are used by the metric store to maintain extra
	// info about the metric which aren't directly useful to the prometheus
	// protocol.
	key   string
	sum   float64
	count uint64
}

func makeMetric(m stats.Metric, v float64, t time.Time) Metric {
	metric := Metric{
		Name:   sanitize(m.Name()),
		Type:   m.Type(),
		Help:   m.Help(),
		Value:  v,
		Time:   t,
		Labels: makeLabels(m.Tags()),
	}
	metric.key = metric.Name + metric.Labels.String()
	metric.sum = metric.Value
	metric.count = 1
	return metric
}

func (m *Metric) apply(v float64) {
	switch m.Type {
	case "counter":
		m.Value += v
	default:
		m.Value = v
	}
	m.sum += v
	m.count++
}

type MetricWriter interface {
	WriteMetric(Metric) error
}

func NewTextWriter(w io.Writer) MetricWriter {
	return &textWriter{Writer: w}
}

type textWriter struct {
	io.Writer
	mutex    sync.Mutex
	lastName string
	count    int
}

func (w *textWriter) WriteMetric(m Metric) (err error) {
	w.mutex.Lock()

	if w.lastName != m.Name {
		w.lastName = m.Name

		if w.count != 0 {
			w.writeNewLine()
		}

		w.writeHelp(m)
		w.writeType(m)
	}

	switch m.Type {
	case "histogram":
		cnt := m
		cnt.Name += "_count"
		cnt.Value = float64(cnt.count)

		sum := m
		sum.Name += "_sum"
		sum.Value = sum.sum

		w.writeMetric(cnt)
		w.writeMetric(sum)

	default:
		w.writeMetric(m)
	}

	w.count++
	w.mutex.Unlock()
	return
}

func (w *textWriter) writeHelp(m Metric) {
	if len(m.Help) != 0 {
		fmt.Fprintf(w, "# HELP %s %s\n", m.Name, escape(m.Help))
	}
}

func (w *textWriter) writeType(m Metric) {
	if len(m.Type) != 0 {
		fmt.Fprintf(w, "# TYPE %s %s\n", m.Name, m.Type)
	}
}

func (w *textWriter) writeMetric(m Metric) {
	fmt.Fprintf(w, "%s%v %g", m.Name, m.Labels, m.Value)

	if m.Time != (time.Time{}) {
		fmt.Fprintf(w, " %d", m.Time.UnixNano()/1000000)
	}

	w.writeNewLine()
}

func (w *textWriter) writeNewLine() {
	io.WriteString(w, "\n")
}

type metricStore struct {
	mutex   sync.RWMutex
	types   map[string]string
	metrics map[string]*Metric
}

func newMetricStore() *metricStore {
	return &metricStore{
		types:   make(map[string]string, 100),
		metrics: make(map[string]*Metric, 100),
	}
}

func (s *metricStore) insert(m Metric) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if t, ok := s.types[m.Name]; ok && t != m.Type {
		return fmt.Errorf(
			"%s was previously registered with type '%s' and therefore cannot be converted to type '%s'",
			m.Name, t, m.Type,
		)
	}

	if x := s.metrics[m.key]; x == nil {
		s.types[m.Name] = m.Type
		s.metrics[m.key] = &m
	} else {
		x.apply(m.Value)
		x.Time = m.Time
	}

	return
}

func (s *metricStore) expire(deadline time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for k, m := range s.metrics {
		if m.Time.Before(deadline) {
			delete(s.metrics, k)
		}
	}
}

func (s *metricStore) snapshot() []Metric {
	m := s.unsortedSnapshot()
	sort.Sort(metrics(m))
	return m
}

func (s *metricStore) unsortedSnapshot() []Metric {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	m := make([]Metric, len(s.metrics))
	i := 0

	for _, x := range s.metrics {
		m[i] = *x
		i++
	}

	return m
}

type metrics []Metric

func (list metrics) Len() int { return len(list) }

func (list metrics) Swap(i int, j int) { list[i], list[j] = list[j], list[i] }

func (list metrics) Less(i int, j int) bool { return list[i].key < list[j].key }
