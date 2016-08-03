package stats

import (
	"bytes"
	"testing"
	"time"
)

func TestMultiBackend(t *testing.T) {
	b1 := &bytes.Buffer{}
	b2 := &bytes.Buffer{}

	b1.Grow(4096)
	b2.Grow(4096)

	b := MultiBackend(
		NewBackend(b1),
		NewBackend(b2),
	)

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

	m1.Set(1)
	m1.Set(42)
	m2.Add(-10)
	m1.Set(0)
	m3.Observe(time.Second)

	c.Close()

	const ref = `{"type":"gauge","name":"test.events.quantity","value":1,"tags":{"hello":"world"}}
{"type":"gauge","name":"test.events.quantity","value":42,"tags":{"hello":"world"}}
{"type":"counter","name":"test.events.count","value":-10,"tags":{"hello":"world","extra":"tag"}}
{"type":"gauge","name":"test.events.quantity","value":0,"tags":{"hello":"world"}}
{"type":"histogram","name":"test.events.duration","value":1,"tags":{"hello":"world"}}
`

	if s := b1.String(); s != ref {
		t.Errorf("invalid string in first backend: %s", s)
	}

	if s := b2.String(); s != ref {
		t.Errorf("invalid string in second backend: %s", s)
	}
}
