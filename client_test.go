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
	m4 := c.Timer("events.duration").Start()

	m1.Set(1)
	m1.Set(42)
	m2.Set(10)
	m1.Set(0)
	m3.Observe(1)

	m4.Stamp("a")
	m4.Stamp("b")
	m4.Stamp("c")
	m4.Stop()

	c.Close()

	if !reflect.DeepEqual(b.Events, []Event{
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 42,
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "counter",
			Name:  "test.events.count",
			Value: 10,
			Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 0,
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "a"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "b"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "c"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: 4,
			Tags:  Tags{{"hello", "world"}},
		},
	}) {
		t.Errorf("invalid events: %#v", b.Events)
	}
}
