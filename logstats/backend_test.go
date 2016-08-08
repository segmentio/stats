package logstats

import (
	"bytes"
	"log"
	"testing"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	b := &bytes.Buffer{}
	c := stats.NewClient("log", NewBackend(log.New(b, "", 0)), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").Set(1)
	c.Counter("events.count").Add(1)
	c.Histogram("events.duration").Observe(1)
	c.Close()

	if s := b.String(); s != `gauge log.events.level [hello=world] 1
counter log.events.count [hello=world] 1
histogram log.events.duration [hello=world] 1
` {
		t.Errorf("invalid logs: %s", s)
	}
}
