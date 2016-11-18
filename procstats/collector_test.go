package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestCollector(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	c := StartCollectorWith(Config{
		CollectInterval: 100 * time.Microsecond,
		Collector: MultiCollector(
			NewGoMetrics(engine),
		),
	})

	// Let the collector do a few runs.
	time.Sleep(time.Millisecond)
	c.Close()

	// Let the engine process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.State()

	if len(metrics) == 0 {
		t.Error("no metrics were reported by the stats collector")
	}
}

func TestCollectorCloser(t *testing.T) {
	c := StartCollector(nil)

	if err := c.Close(); err != nil {
		t.Error("unexpected error reported when closing a collector:", err)
	}
}
