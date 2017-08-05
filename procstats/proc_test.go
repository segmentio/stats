package procstats

import (
	"os"
	"testing"

	"github.com/segmentio/stats"
)

func TestProcMetrics(t *testing.T) {
	h := &handler{}
	e := stats.NewEngine("", h)

	proc := NewProcMetricsWith(e, os.Getpid())
	proc.Collect()

	if len(h.measures) == 0 {
		t.Error("no measures were reported by the stats collector")
	}

	for _, m := range h.measures {
		t.Log(m)
	}
}
