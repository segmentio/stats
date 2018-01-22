package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestCollector(t *testing.T) {
	h := &statstest.Handler{}
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

	if len(h.Measures()) == 0 {
		t.Error("no measures were reported by the stats collector")
	}

	for _, m := range h.Measures() {
		t.Log(m)
	}
}

func TestCollectorCloser(t *testing.T) {
	c := StartCollector(nil)

	if err := c.Close(); err != nil {
		t.Error("unexpected error reported when closing a collector:", err)
	}
}
