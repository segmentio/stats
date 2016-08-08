package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	now := time.Unix(1, 0)

	b := &EventBackend{}
	c := NewClientWith(Config{
		Backend: b,
		Scope:   "test",
		Tags:    Tags{{"hello", "world"}},
		Now:     func() time.Time { return now },
	})

	m1 := c.Gauge("events.quantity")
	m2 := c.Counter("events.count", Tag{"extra", "tag"})
	m3 := c.Histogram("events.seconds")
	m4 := c.Timer("events.seconds").Start()

	m1.Set(1)
	m1.Set(42)
	m2.Add(10)
	m1.Set(0)
	m3.Observe(1)

	m4.StampAt("a", now.Add(1*time.Second))
	m4.StampAt("b", now.Add(2*time.Second))
	m4.StampAt("c", now.Add(3*time.Second))
	m4.StopAt(now.Add(4 * time.Second))

	c.Close()

	m4.Stamp("d")
	m4.Stop()

	events := []Event{
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 42,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
		Event{
			Type:  "counter",
			Name:  "test.events.count",
			Value: 10,
			Tags:  Tags{{"hello", "world"}, {"extra", "tag"}},
			Time:  now,
		},
		Event{
			Type:  "gauge",
			Name:  "test.events.quantity",
			Value: 0,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 1,
			Tags:  Tags{{"hello", "world"}},
			Time:  now,
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "a"}},
			Time:  now.Add(1 * time.Second),
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "b"}},
			Time:  now.Add(2 * time.Second),
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 1,
			Tags:  Tags{{"hello", "world"}, {"stamp", "c"}},
			Time:  now.Add(3 * time.Second),
		},
		Event{
			Type:  "histogram",
			Name:  "test.events.seconds",
			Value: 4,
			Tags:  Tags{{"hello", "world"}},
			Time:  now.Add(4 * time.Second),
		},
	}

	if !reflect.DeepEqual(b.Events, events) {
		for i := range events {
			e1 := b.Events[i]
			e2 := events[i]

			if !reflect.DeepEqual(e1, e2) {
				t.Errorf("#%d:\n- %#v\n- %#v", i, e1, e2)
			}
		}
	}
}

func TestClientClose(t *testing.T) {
	client := NewClient("app", &EventBackend{})
	client.Close()
}
