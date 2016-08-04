package logrus_stats

import (
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/segmentio/stats"
)

type hook struct {
	entries []*logrus.Entry
}

func (h *hook) Levels() []logrus.Level {
	return []logrus.Level{logrus.InfoLevel}
}

func (h *hook) Fire(e *logrus.Entry) {
	h.entries = append(h.entries, e)
}

func TestBackend(t *testing.T) {
	h := &hook{}
	c := stats.NewClient(
		"logrus",
		NewBackend(&logrus.Logger{
			Out:   w,
			Hooks: logrus.LevelHooks{logrus.InfoLevel: h},
		}),
		stats.Tag{
			Name:  "hello",
			Value: "world",
		},
	)

	c.Gauge(stats.Opts{Name: "events", Unit: "level"}).Set(1)
	c.Counter(stats.Opts{Name: "events", Unit: "count"}).Add(1)
	c.Histogram(stats.Opts{Name: "events", Unit: "duration"}).Observe(time.Second)
	c.Close()

	if n := len(h.Entries); n != 3 {
		t.Errorf("invalid number of log entries:", n)
	}

	for _, e := range h.Entries {
		if _, ok := e.Data["metric"]; !ok {
			t.Errorf("missing 'metric' in log entry")
		}
	}
}
