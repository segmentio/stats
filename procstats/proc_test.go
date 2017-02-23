package procstats

import (
	"os"
	"strings"
	"testing"

	"github.com/segmentio/stats"
)

func TestProcMetrics(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine("")
	e.Register(h)

	proc := NewProcMetricsWith(e, os.Getpid())
	proc.Collect()

	if len(h.metrics) == 0 {
		t.Error("no metrics were reported by the stats collector")
	}

	for _, m := range h.metrics {
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
