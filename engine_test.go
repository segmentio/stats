package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestEngineRegister(t *testing.T) {
	h1 := &handler{}
	h2 := &handler{}
	h3 := &handler{}

	eng := NewEngineWith("E")
	eng.Register(h1)
	eng.Register(h2)
	eng.Register(h3)

	if name := eng.Name(); name != "E" {
		t.Error("bad engine name:", name)
	}

	if tags := eng.Tags(); len(tags) != 0 {
		t.Error("bad engine tags:", tags)
	}

	if handlers := eng.Handlers(); !reflect.DeepEqual(handlers, []Handler{h1, h2, h3}) {
		t.Error("bad handlers:", handlers)
	}
}

func TestEngineClone(t *testing.T) {
	h1 := &handler{}
	h2 := &handler{}
	h3 := &handler{}

	eng1 := NewEngineWith("E")
	eng1.Register(h1)
	eng1.Register(h2)

	eng2 := eng1.Clone(
		Tag{"A", "1"},
		Tag{"B", "2"},
		Tag{"C", "3"},
	)
	eng2.Register(h3)

	if name := eng2.Name(); name != "E" {
		t.Error("bad engine name:", name)
	}

	if tags := eng2.Tags(); !reflect.DeepEqual(tags, []Tag{{"A", "1"}, {"B", "2"}, {"C", "3"}}) {
		t.Error("bad engine tags:", tags)
	}

	if handlers := eng2.Handlers(); !reflect.DeepEqual(handlers, []Handler{h1, h2, h3}) {
		t.Error("bad handlers:", handlers)
	}
}

func TestEngineFlush(t *testing.T) {
	h1 := &handler{}
	h2 := &handler{}
	h3 := &handler{}

	eng := NewEngineWith("E")
	eng.Register(h1)
	eng.Register(h2)
	eng.Register(h3)

	eng.Flush()

	for i, h := range []*handler{h1, h2, h3} {
		if h.flushed != 1 {
			t.Error("handler at index", i, "was not flushed")
		}
	}
}

func TestEngineAdd(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E", Tag{"base", "tag"})
	e.Register(h)

	e.Incr("A")
	e.Add("B", 2)
	e.Add("C", 3, Tag{"extra", "tag"})

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      CounterType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      CounterType,
			Namespace: "E",
			Name:      "B",
			Value:     2,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      CounterType,
			Namespace: "E",
			Name:      "C",
			Value:     3,
			Tags:      []Tag{{"base", "tag"}, {"extra", "tag"}},
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestEngineSet(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E", Tag{"base", "tag"})
	e.Register(h)

	e.Set("A", 1)
	e.Set("B", 2)
	e.Set("C", 3, Tag{"extra", "tag"})

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "B",
			Value:     2,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "C",
			Value:     3,
			Tags:      []Tag{{"base", "tag"}, {"extra", "tag"}},
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestEngineObserve(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E", Tag{"base", "tag"})
	e.Register(h)

	e.Observe("A", 1)
	e.Observe("B", 2)
	e.Observe("C", 3, Tag{"extra", "tag"})

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "B",
			Value:     2,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "C",
			Value:     3,
			Tags:      []Tag{{"base", "tag"}, {"extra", "tag"}},
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestEngineObserveDuration(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E", Tag{"base", "tag"})
	e.Register(h)

	e.ObserveDuration("A", 1*time.Second)
	e.ObserveDuration("B", 2*time.Second)
	e.ObserveDuration("C", 3*time.Second, Tag{"extra", "tag"})

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "B",
			Value:     2,
			Tags:      []Tag{{"base", "tag"}},
		},
		{
			Type:      HistogramType,
			Namespace: "E",
			Name:      "C",
			Value:     3,
			Tags:      []Tag{{"base", "tag"}, {"extra", "tag"}},
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestEngineCounter(t *testing.T) {
	e := NewEngineWith("E", Tag{"base", "tag"})
	c := e.Counter("C", Tag{"extra", "tag"})

	if name := c.Name(); name != "C" {
		t.Error("bad counter name:", name)
	}

	if tags := c.Tags(); !reflect.DeepEqual(tags, []Tag{{"extra", "tag"}}) {
		t.Error("bad counter tags:", tags)
	}
}

func TestEngineGauge(t *testing.T) {
	e := NewEngineWith("E", Tag{"base", "tag"})
	g := e.Gauge("G", Tag{"extra", "tag"})

	if name := g.Name(); name != "G" {
		t.Error("bad gauge name:", name)
	}

	if tags := g.Tags(); !reflect.DeepEqual(tags, []Tag{{"extra", "tag"}}) {
		t.Error("bad gauge tags:", tags)
	}
}

func TestEngineHistogram(t *testing.T) {
	e := NewEngineWith("E", Tag{"base", "tag"})
	h := e.Histogram("H", Tag{"extra", "tag"})

	if name := h.Name(); name != "H" {
		t.Error("bad histogram name:", name)
	}

	if tags := h.Tags(); !reflect.DeepEqual(tags, []Tag{{"extra", "tag"}}) {
		t.Error("bad histogram tags:", tags)
	}
}

func TestEngineTimer(t *testing.T) {
	e := NewEngineWith("E", Tag{"base", "tag"})
	h := e.Timer("H", Tag{"extra", "tag"})

	if name := h.Name(); name != "H" {
		t.Error("bad timer name:", name)
	}

	if tags := h.Tags(); !reflect.DeepEqual(tags, []Tag{{"extra", "tag"}}) {
		t.Error("bad timer tags:", tags)
	}
}
