package stats

import (
	"reflect"
	"testing"
)

func TestHistogramIncr(t *testing.T) {
	h := &handler{}
	e := NewEngine("E")
	e.Register(h)

	m := e.Histogram("A")
	m.Observe(1)

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestHistogramSet(t *testing.T) {
	h := &handler{}
	e := NewEngine("E")
	e.Register(h)

	m := e.Histogram("A")
	m.Observe(1)
	m.Observe(0.5)

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Value:     0.5,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestHistogramWithTags(t *testing.T) {
	e := NewEngine("E")
	c1 := e.Histogram("A", Tag{"base", "tag"})
	c2 := c1.WithTags(Tag{"extra", "tag"})

	if name := c2.Name(); name != "A" {
		t.Error("bad histogram name:", name)
	}

	if tags := c2.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad histogram tags:", tags)
	}
}

func BenchmarkHistogram(b *testing.B) {
	e := NewEngine("E")

	b.Run("Observe", func(b *testing.B) {
		h := e.Histogram("A")
		for i := 0; i != b.N; i++ {
			h.Observe(float64(i))
		}
	})
}
