package stats

import (
	"reflect"
	"testing"
)

func TestClockStart(t *testing.T) {
	h := &handler{}
	e := NewEngine("E")
	e.Register(h)

	m := e.Timer("A")
	c := m.Start()
	c.Stamp("1")
	c.Stamp("2")
	c.Stamp("3")
	c.Stop()

	for i := range h.metrics {
		if h.metrics[i].Value == 0 {
			t.Error("clock time value should not be zero")
		}
		h.metrics[i].Value = 0 // unpredictable
	}

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Tags:      []Tag{{"stamp", "1"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Tags:      []Tag{{"stamp", "2"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Tags:      []Tag{{"stamp", "3"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Tags:      []Tag{{"stamp", "total"}},
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestClockWithTags(t *testing.T) {
	e := NewEngine("E")
	m := e.Timer("A", Tag{"base", "tag"})
	c1 := m.Start()
	c2 := c1.WithTags(Tag{"extra", "tag"})

	if name := c2.Name(); name != "A" {
		t.Error("bad clock name:", name)
	}

	if tags := c2.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad clock tags:", tags)
	}
}
