package stats

import (
	"bytes"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	now := time.Now()

	b := &bytes.Buffer{}
	b.Grow(4096)

	c := NewClient("test", NewBackend(b), Tag{"hello", "world"})

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
	m2.Add(-10)
	m1.Set(0)
	m3.Observe(time.Second)

	m4.Lap(now.Add(1*time.Second), "a")
	m4.Lap(now.Add(2*time.Second), "b")
	m4.Lap(now.Add(3*time.Second), "c")
	m4.Stop(now.Add(4 * time.Second))

	c.Close()
	s := b.String()

	if s != `{"type":"gauge","name":"test.events.quantity","value":1,"tags":{"hello":"world"}}
{"type":"gauge","name":"test.events.quantity","value":42,"tags":{"hello":"world"}}
{"type":"counter","name":"test.events.count","value":-10,"tags":{"hello":"world","extra":"tag"}}
{"type":"gauge","name":"test.events.quantity","value":0,"tags":{"hello":"world"}}
{"type":"histogram","name":"test.events.duration","value":1,"tags":{"hello":"world"}}
{"type":"histogram","name":"test.events.duration","value":1,"tags":{"hello":"world","lap":"a"}}
{"type":"histogram","name":"test.events.duration","value":1,"tags":{"hello":"world","lap":"b"}}
{"type":"histogram","name":"test.events.duration","value":1,"tags":{"hello":"world","lap":"c"}}
{"type":"histogram","name":"test.events.duration","value":4,"tags":{"hello":"world"}}
` {
		t.Errorf("invalid string: %s", s)
	}
}
