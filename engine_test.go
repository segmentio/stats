package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	engine := NewDefaultEngine()

	a := engine.Counter("A")
	b := engine.Gauge("B")
	c := engine.Gauge("C", Tag{"context", "test"})

	a.Add(1)
	b.Set(2)
	c.Sub(3)

	// Give a bit of time for the engine to update its state.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.Metrics()
	sortMetrics(metrics)

	if !reflect.DeepEqual(metrics, []Metric{
		Metric{
			Type:    CounterType,
			Key:     "A?",
			Name:    "A",
			Tags:    nil,
			Value:   1,
			Version: 1,
		},
		Metric{
			Type:    GaugeType,
			Key:     "B?",
			Name:    "B",
			Tags:    nil,
			Value:   2,
			Version: 1,
		},
		Metric{
			Type:    GaugeType,
			Key:     "C?context=test",
			Name:    "C",
			Tags:    []Tag{{"context", "test"}},
			Value:   -3,
			Version: 1,
		},
	}) {
		t.Errorf("bad engine state: %#v", metrics)
	}

	engine.Close()
}
