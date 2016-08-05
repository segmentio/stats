package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	now := time.Now()

	b := &EventBackend{}
	c := NewClientWith(Config{
		Backend: b,
		Scope:   "test",
		Tags:    Tags{{"hello", "world"}},
		Now: func() time.Time {
			t := now
			now = now.Add(time.Second)
			return t
		},
	})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.duration")
	m4 := c.Timer("events.duration")

	m1.Set(1)
	m1.Set(42)
	m2.Add(10)
	m1.Set(0)
	m3.Observe(time.Second)

	m4.Lap("a")
	m4.Lap("b")
	m4.Lap("c")
	m4.Stop()

	c.Close()

	if !reflect.DeepEqual(b.Events, []Event{
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: float64(1),
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: float64(42),
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "counter",
			Name:  "test.events.count",
			Value: float64(10),
			Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: float64(0),
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: time.Second,
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: time.Second,
			Tags:  Tags{{"hello", "world"}, {"lap", "a"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: time.Second,
			Tags:  Tags{{"hello", "world"}, {"lap", "b"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: time.Second,
			Tags:  Tags{{"hello", "world"}, {"lap", "c"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 4 * time.Second,
			Tags:  Tags{{"hello", "world"}},
		},
	}) {
		t.Errorf("invalid events: %#v", b.Events)
	}
}
