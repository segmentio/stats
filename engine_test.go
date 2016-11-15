package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	engine := NewEngine(EngineConfig{
		Prefix: "test",
		Tags:   []Tag{{"hello", "world"}},
	})

	a := engine.Counter("A")
	b := engine.Gauge("B")
	c := engine.Gauge("C", Tag{"context", "test"})

	a.Add(1)
	b.Set(2)
	c.Sub(3)

	// Give a bit of time for the engine to update its state.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.State()
	sortMetrics(metrics)

	if !reflect.DeepEqual(metrics, []Metric{
		Metric{
			Type:  CounterType,
			Key:   "test.A?hello=world",
			Name:  "test.A",
			Tags:  []Tag{{"hello", "world"}},
			Value: 1,
			Count: 1,
		},
		Metric{
			Type:  GaugeType,
			Key:   "test.B?hello=world",
			Name:  "test.B",
			Tags:  []Tag{{"hello", "world"}},
			Value: 2,
			Count: 1,
		},
		Metric{
			Type:  GaugeType,
			Key:   "test.C?context=test&hello=world",
			Name:  "test.C",
			Tags:  []Tag{{"context", "test"}, {"hello", "world"}},
			Value: -3,
			Count: 1,
		},
	}) {
		t.Errorf("bad engine state: %#v", metrics)
	}

	engine.Close()
}
