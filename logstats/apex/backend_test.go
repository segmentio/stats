package apexstats

import (
	"testing"

	"github.com/apex/log"
	"github.com/apex/log/handlers/memory"
	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	h := &memory.Handler{}
	c := stats.NewClient("apex", NewBackend(&log.Logger{Handler: h}), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").Set(1)
	c.Counter("events.count").Add(1)
	c.Histogram("events.duration").Observe(1)
	c.Close()

	if n := len(h.Entries); n != 3 {
		t.Errorf("invalid number of log entries: %d", n)
	}

	for _, e := range h.Entries {
		if _, ok := e.Fields["metric"]; !ok {
			t.Error("missing 'metric' in log entry")
		}
	}
}
