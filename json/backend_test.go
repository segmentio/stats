package json_stats

import (
	"bytes"
	"testing"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	b := &bytes.Buffer{}
	c := stats.NewClient("log", NewBackend(b), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").Set(1)
	c.Counter("events.count").Add(1)
	c.Histogram("events.duration").Observe(1)
	c.Close()

	if s := b.String(); s != `{"type":"gauge","name":"log.events.level","value":1,"tags":{"hello":"world"}}
{"type":"counter","name":"log.events.count","value":1,"tags":{"hello":"world"}}
{"type":"histogram","name":"log.events.duration","value":1,"tags":{"hello":"world"}}
` {
		t.Errorf("invalid json: %s", s)
	}
}
