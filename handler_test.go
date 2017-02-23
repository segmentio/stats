package stats

import (
	"reflect"
	"testing"
	"time"
)

type handler struct {
	metrics []Metric
	flushed int
}

func (h *handler) HandleMetric(m *Metric) {
	c := *m
	c.Tags = copyTags(c.Tags)
	c.Time = time.Time{} // discard because it's unpredicatable
	h.metrics = append(h.metrics, c)
}

func (h *handler) Flush() {
	h.flushed++
}

func TestHandlerFunc(t *testing.T) {
	metrics := []Metric{
		{
			Type:  CounterType,
			Name:  "A",
			Value: 1,
		},
		{
			Type:  GaugeType,
			Name:  "B",
			Value: 2,
		},
		{
			Type:  HistogramType,
			Name:  "C",
			Value: 3,
		},
	}

	r := []Metric{}
	f := HandlerFunc(func(m *Metric) {
		r = append(r, *m)
	})

	for i := range metrics {
		f.HandleMetric(&metrics[i])
	}

	if !reflect.DeepEqual(r, metrics) {
		t.Error("bad metrics reported:", r)
	}
}
