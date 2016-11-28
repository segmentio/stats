package procstats

import (
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestProcMetrics(t *testing.T) {
	engine := stats.NewDefaultEngine()
	defer engine.Close()

	proc := NewProcMetrics(engine)
	proc.Collect()

	// Let the engine process the metrics.
	time.Sleep(10 * time.Millisecond)

	metrics := engine.State()

	if len(metrics) == 0 {
		t.Error("no metrics were reported by the stats collector")
	}

	for _, m := range metrics {
		switch {
		case strings.HasPrefix(m.Name, "procstats.test.cpu."):
		case strings.HasPrefix(m.Name, "procstats.test.memory."):
		case strings.HasPrefix(m.Name, "procstats.test.files."):
		case strings.HasPrefix(m.Name, "procstats.test.thread."):
		default:
			t.Error("invalid metric name:", m.Name)
		}
	}
}
