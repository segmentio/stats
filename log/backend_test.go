package log_stats

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	b := &bytes.Buffer{}
	c := stats.NewClient("log", NewBackend(log.New(b, "", 0)), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge(stats.Opts{Name: "events", Unit: "level", Help: "yay!"}).Set(1)
	c.Counter(stats.Opts{Name: "events", Unit: "count"}).Add(1)
	c.Histogram(stats.Opts{Name: "events", Unit: "duration"}).Observe(time.Second)
	c.Close()

	if s := b.String(); s != `gauge log.events.level [hello=world] 1/1 (yay!)
counter log.events.count [hello=world] 1/1 ()
histogram log.events.duration [hello=world] 1s/1 ()
` {
		t.Errorf("invalid logs: %s", s)
	}
}
