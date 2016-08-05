package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestBackendFunc(t *testing.T) {
	e := []Event{}
	b := BackendFunc(func(x Event) { e = append(e, x) })
	c := NewClient("test", b, Tag{"hello", "world"})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.duration")

	m1.Set(1)
	m2.Add(1)
	m3.Observe(time.Second)

	c.Close()

	if !reflect.DeepEqual(e, []Event{
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: float64(1),
			Tags:  Tags{{"hello", "world"}},
		},
		Event{
			Type:  "counter",
			Name:  "test.events.count",
			Value: float64(1),
			Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.duration",
			Value: time.Second,
			Tags:  Tags{{"hello", "world"}},
		},
	}) {
		t.Errorf("invalid events: %#v", e)
	}
}

func TestMultiBackend(t *testing.T) {
	b := []*EventBackend{&EventBackend{}, &EventBackend{}}
	c := NewClient("test", MultiBackend(b[0], b[1]), Tag{"hello", "world"})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.duration")

	m1.Set(1)
	m2.Add(1)
	m3.Observe(time.Second)

	c.Close()

	for _, e := range b {
		if !reflect.DeepEqual(e.Events, []Event{
			Event{
				Type:  "gauge",
				Name:  "test.events.quantity",
				Value: float64(1),
				Tags:  Tags{{"hello", "world"}},
			},
			Event{
				Type:  "counter",
				Name:  "test.events.count",
				Value: float64(1),
				Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
			},
			Event{
				Type:  "histogram",
				Name:  "test.events.duration",
				Value: time.Second,
				Tags:  Tags{{"hello", "world"}},
			},
		}) {
			t.Errorf("invalid events: %#v", e.Events)
		}
	}
}
