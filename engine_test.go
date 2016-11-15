package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	engine := NewDefaultEngine()

	a := engine.Counter("A")
	b := engine.Counter("B")
	c := engine.Counter("C", Tag{"context", "test"})

	a.Add(1)
	b.Add(2)
	c.Add(3)

	// Give a bit of time for the engine to update its state.
	time.Sleep(100 * time.Millisecond)

	if metrics := engine.Metrics(); !reflect.DeepEqual(metrics, []Metric{
		Metric{
			Type:    CounterType,
			Key:     "A?",
			Name:    "A",
			Tags:    nil,
			Value:   1,
			Version: 1,
		},
		Metric{
			Type:    CounterType,
			Key:     "B?",
			Name:    "B",
			Tags:    nil,
			Value:   2,
			Version: 1,
		},
		Metric{
			Type:    CounterType,
			Key:     "C?context=test",
			Name:    "C",
			Tags:    []Tag{{"context", "test"}},
			Value:   3,
			Version: 1,
		},
	}) {
		t.Errorf("bad engine state: %#v", metrics)
	}

	engine.Close()
}
