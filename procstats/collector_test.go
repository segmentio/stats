package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

type handler struct {
	metrics []stats.Metric
}

func (h *handler) HandleMetric(m *stats.Metric) {
	c := *m
	c.Tags = append([]stats.Tag{}, m.Tags...)
	c.Time = time.Time{} // discard because it's unpredicatable
	h.metrics = append(h.metrics, c)
}

func TestCollector(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine()
	e.Register(h)

	c := StartCollectorWith(Config{
		CollectInterval: 100 * time.Microsecond,
		Collector: MultiCollector(
			NewGoMetricsWith(e),
		),
	})

	// Let the collector do a few runs.
	time.Sleep(time.Millisecond)
	c.Close()

	if len(h.metrics) == 0 {
		t.Error("no metrics were reported by the stats collector")
	}
}

func TestCollectorCloser(t *testing.T) {
	c := StartCollector(nil)

	if err := c.Close(); err != nil {
		t.Error("unexpected error reported when closing a collector:", err)
	}
}
