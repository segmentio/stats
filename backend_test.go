package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestBackendFunc(t *testing.T) {
	now := time.Now()

	e := []Event{}
	b := BackendFunc(func(x Event) { e = append(e, x) })
	c := NewClientWith(Config{
		Scope:   "test",
		Backend: b,
		Tags:    Tags{{"hello", "world"}},
		Now:     func() time.Time { return now },
	})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.seconds")

	m1.Set(1)
	m2.Add(1)
	m3.Observe(1)

	c.Close()

	if !reflect.DeepEqual(e, []Event{
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
		Event{
			Type:  "counter",
			Name:  "test.events.count",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
			Time:  now,
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
	}) {
		t.Errorf("invalid events: %#v", e)
	}
}

func TestMultiBackend(t *testing.T) {
	now := time.Now()

	b := []*EventBackend{&EventBackend{}, &EventBackend{}}
	c := NewClientWith(Config{
		Scope:   "test",
		Backend: MultiBackend(b[0], b[1]),
		Now:     func() time.Time { return now },
		Tags:    Tags{{"hello", "world"}},
	})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.seconds")

	m1.Set(1)
	m2.Add(1)
	m3.Observe(1)

	c.Close()

	for _, e := range b {
		if !reflect.DeepEqual(e.Events, []Event{
			Event{
				Type:  "gauge",
				Name:  "test.events.quantity",
				Value: 1,
				Tags:  Tags{{"hello", "world"}},
				Time:  now,
			},
			Event{
				Type:  "counter",
				Name:  "test.events.count",
				Value: 1,
				Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
				Time:  now,
			},
			Event{
				Type:  "histogram",
				Name:  "test.events.seconds",
				Value: 1,
				Tags:  Tags{{"hello", "world"}},
				Time:  now,
			},
		}) {
			t.Errorf("invalid events: %#v", e.Events)
		}
	}
}
