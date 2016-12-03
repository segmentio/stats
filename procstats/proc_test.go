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

	metrics, _ := engine.State(0)

	if len(metrics) == 0 {
		t.Error("no metrics were reported by the stats collector")
	}

	for _, m := range metrics {
		switch {
		case strings.HasPrefix(m.Name, "cpu."):
		case strings.HasPrefix(m.Name, "memory."):
		case strings.HasPrefix(m.Name, "files."):
		case strings.HasPrefix(m.Name, "thread."):
		default:
			t.Error("invalid metric name:", m.Name)
		}
	}
}
