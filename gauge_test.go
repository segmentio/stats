package stats

import (
	"reflect"
	"testing"
)

func TestGaugeIncr(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E")
	e.Register(h)

	g := NewGauge(e, "A")
	g.Incr()

	if v := g.Value(); v != 1 {
		t.Error("bad value:", v)
	}

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestGaugeDecr(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E")
	e.Register(h)

	g := NewGauge(e, "A")
	g.Decr()

	if v := g.Value(); v != -1 {
		t.Error("bad value:", v)
	}

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     -1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestGaugeAdd(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E")
	e.Register(h)

	g := NewGauge(e, "A")
	g.Add(0.5)
	g.Add(0.5)

	if v := g.Value(); v != 1 {
		t.Error("bad value:", v)
	}

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     0.5,
		},
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestGaugeSet(t *testing.T) {
	h := &handler{}
	e := NewEngineWith("E")
	e.Register(h)

	g := NewGauge(e, "A")
	g.Set(1)
	g.Set(0.5)

	if v := g.Value(); v != 0.5 {
		t.Error("bad value:", v)
	}

	if !reflect.DeepEqual(h.metrics, []Metric{
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     1,
		},
		{
			Type:      GaugeType,
			Namespace: "E",
			Name:      "A",
			Value:     0.5,
		},
	}) {
		t.Error("bad metrics:", h.metrics)
	}
}

func TestGaugeClone(t *testing.T) {
	e := NewEngineWith("E")
	c1 := NewGauge(e, "A", T("base", "tag"))
	c2 := c1.Clone(T("extra", "tag"))

	if name := c2.Name(); name != "A" {
		t.Error("bad gauge name:", name)
	}

	if tags := c2.Tags(); !reflect.DeepEqual(tags, []Tag{{"base", "tag"}, {"extra", "tag"}}) {
		t.Error("bad gauge tags:", tags)
	}
}
