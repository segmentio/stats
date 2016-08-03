package stats

import (
	"bytes"
	"testing"
)

func TestClient(t *testing.T) {
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

	m1.Set(1)
	m1.Set(42)
	m2.Add(-10)
	m1.Set(0)

	c.Close()
	s := b.String()

	if s != `{"type":"gauge","name":"test.events.quantity","value":1,"tags":{"hello":"world"}}
{"type":"gauge","name":"test.events.quantity","value":42,"tags":{"hello":"world"}}
{"type":"counter","name":"test.events.count","value":-10,"tags":{"extra":"tag","hello":"world"}}
{"type":"gauge","name":"test.events.quantity","value":0,"tags":{"hello":"world"}}
` {
		t.Errorf("invalid string: %s", s)
	}
}
