package logstats

import (
	"bytes"
	"log"
	"testing"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	b := &bytes.Buffer{}
	c := stats.NewClient(NewBackend(log.New(b, "", 0)), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").Set(1)
	c.Counter("events.count").Add(1)
	c.Histogram("events.seconds").Observe(1)
	c.Close()

	if s := b.String(); s != `gauge logstats.test.events.level [hello=world] 1/1
counter logstats.test.events.count [hello=world] 1/1
histogram logstats.test.events.seconds [hello=world] 1/1
` {
		t.Errorf("invalid logs: %s", s)
	}
}
