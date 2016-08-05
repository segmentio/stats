package json_stats

import (
	"bytes"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	b := &bytes.Buffer{}
	c := stats.NewClient("log", NewBackend(b), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge(stats.Opts{Name: "events", Unit: "level", Help: "yay!"}).Set(1)
	c.Counter(stats.Opts{Name: "events", Unit: "count"}).Add(1)
	c.Histogram(stats.Opts{Name: "events", Unit: "duration"}).Observe(time.Second)
	c.Close()

	if s := b.String(); s != `{"type":"gauge","name":"log.events.level","help":"yay!","value":1,"tags":{"hello":"world"}}
{"type":"counter","name":"log.events.count","value":1,"tags":{"hello":"world"}}
{"type":"histogram","name":"log.events.duration","value":1000000000,"tags":{"hello":"world"}}
` {
		t.Errorf("invalid json: %s", s)
	}
}
