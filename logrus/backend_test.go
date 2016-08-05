package logrus_stats

import (
	"bytes"
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

func (h *hook) Fire(e *logrus.Entry) error {
	h.entries = append(h.entries, e)
	return nil
}

func TestBackend(t *testing.T) {
	h := &hook{}
	b := &bytes.Buffer{}
	c := stats.NewClient(
		"logrus",
		NewBackend(&logrus.Logger{
			Out:       b,
			Hooks:     logrus.LevelHooks{logrus.InfoLevel: []logrus.Hook{h}},
			Formatter: &logrus.JSONFormatter{},
			Level:     logrus.DebugLevel,
		}),
		stats.Tag{
			Name:  "hello",
			Value: "world",
		},
	)

	c.Gauge("events.level").Set(1)
	c.Counter("events.count").Add(1)
	c.Histogram("events.duration").Observe(time.Second)
	c.Close()

	if n := len(h.entries); n != 3 {
		t.Errorf("invalid number of log entries: %d", n)
	}

	for _, e := range h.entries {
		if _, ok := e.Data["metric"]; !ok {
			t.Error("missing 'metric' in log entry")
		}
	}
}
