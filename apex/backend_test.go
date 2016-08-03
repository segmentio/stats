package apex_stats

import (
	"testing"
	"time"

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

	c.Gauge(stats.Opts{Name: "events", Unit: "level"}).Set(1)
	c.Counter(stats.Opts{Name: "events", Unit: "count"}).Add(1)
	c.Histogram(stats.Opts{Name: "events", Unit: "duration"}).Observe(time.Second)
	c.Close()

	if n := len(h.Entries); n != 3 {
		t.Errorf("invalid number of log entries:", n)
	}

	for _, e := range h.Entries {
		if _, ok := e.Fields["metric"]; !ok {
			t.Errorf("missing 'metric' in log entry")
		}
	}
}
