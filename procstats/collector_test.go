package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

type handler struct {
	measures []stats.Measure
}

func (h *handler) HandleMeasures(time time.Time, measures ...stats.Measure) {
	for _, m := range measures {
		h.measures = append(h.measures, m.Clone())
	}
}

func TestCollector(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine("", h)

	c := StartCollectorWith(Config{
		CollectInterval: 100 * time.Microsecond,
		Collector: MultiCollector(
			NewGoMetricsWith(e),
		),
	})

	// Let the collector do a few runs.
	time.Sleep(time.Millisecond)
	c.Close()

	if len(h.measures) == 0 {
		t.Error("no measures were reported by the stats collector")
	}

	for _, m := range h.measures {
		t.Log(m)
	}
}

func TestCollectorCloser(t *testing.T) {
	c := StartCollector(nil)

	if err := c.Close(); err != nil {
		t.Error("unexpected error reported when closing a collector:", err)
	}
}
