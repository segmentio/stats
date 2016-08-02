package stats

import (
	"bytes"
	"testing"
)

func TestClient(t *testing.T) {
	b := &bytes.Buffer{}
	b.Grow(4096)

	c := NewClient(b)

	t0 := c.NewTracker("A")
	t1 := c.NewTracker("B")

	t0.Incr(NewMetric("test", Tag{"S", "hello"}), Count(1))
	t0.Decr(NewMetric("test", Tag{"S", "world"}), Count(1))
	t1.Track(NewMetric("test", Tag{"S", "!"}), Size(10))

	c.Close()
	s := b.String()

	if s != `{"name":"A.test.count","type":"add","value":1,"tags":{"S":"hello"}}
{"name":"A.test.count","type":"sub","value":1,"tags":{"S":"world"}}
{"name":"B.test.size","type":"set","value":10,"tags":{"S":"!"}}
` {
		t.Errorf("invalid string: %s", s)
	}
}
