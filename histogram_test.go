package stats

import (
	"reflect"
	"testing"
)

func TestHistogramIncr(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E")
	e.Register(h)

	m := NewHistogram(e, "A")
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
	e := NewEngineWith("E")
	e.Register(h)

	m := NewHistogram(e, "A")
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

func TestHistogramClone(t *testing.T) {
	e := NewEngineWith("E")
	c1 := NewHistogram(e, "A", T("base", "tag"))
	c2 := c1.Clone(T("extra", "tag"))

	if name := c2.Name(); name != "A" {
		t.Error("bad histogram name:", name)
	}

	if tags := c2.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad histogram tags:", tags)
	}
}
