package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	now := time.Now()

	b := &EventBackend{}
	c := NewClient("test", b, Tag{"hello", "world"})

	m1 := c.Gauge(Opts{
		Name: "events",
		Unit: "quantity",
	})

	m2 := c.Counter(Opts{
		Name: "events",
		Unit: "count",
		Tags: Tags{{"extra", "tag"}},
	})

	m3 := c.Histogram(Opts{
		Name: "events",
		Unit: "duration",
	})

	m4 := c.Timer(now, Opts{
		Name: "events",
		Unit: "duration",
	})

	m1.Set(1)
	m1.Set(42)
	m2.Add(10)
	m1.Set(0)
	m3.Observe(time.Second)

	m4.Lap(now.Add(1*time.Second), "a")
	m4.Lap(now.Add(2*time.Second), "b")
	m4.Lap(now.Add(3*time.Second), "c")
	m4.Stop(now.Add(4 * time.Second))

	c.Close()

	if !reflect.DeepEqual(b.Events, []Event{
		Event{
			Type:   "gauge",
			Name:   "test.events.quantity",
			Value:  float64(1),
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}},
		},
		Event{
			Type:   "gauge",
			Name:   "test.events.quantity",
			Value:  float64(42),
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}},
		},
		Event{
			Type:   "counter",
			Name:   "test.events.count",
			Value:  float64(10),
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}, {"extra", "tag"}},
		},
		Event{
			Type:   "gauge",
			Name:   "test.events.quantity",
			Value:  float64(0),
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}},
		},
		Event{
			Type:   "histogram",
			Name:   "test.events.duration",
			Value:  time.Second,
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}},
		},
		Event{
			Type:   "histogram",
			Name:   "test.events.duration",
			Value:  time.Second,
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}, {"lap", "a"}},
		},
		Event{
			Type:   "histogram",
			Name:   "test.events.duration",
			Value:  time.Second,
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}, {"lap", "b"}},
		},
		Event{
			Type:   "histogram",
			Name:   "test.events.duration",
			Value:  time.Second,
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}, {"lap", "c"}},
		},
		Event{
			Type:   "histogram",
			Name:   "test.events.duration",
			Value:  4 * time.Second,
			Sample: float64(1),
			Tags:   Tags{{"hello", "world"}},
		},
	}) {
		t.Errorf("invalid events: %#v", b.Events)
	}
}
