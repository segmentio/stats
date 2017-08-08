package procstats

import (
	"os"
	"testing"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestProcMetrics(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	proc := NewProcMetricsWith(e, os.Getpid())
	proc.Collect()

	if len(h.Measures()) == 0 {
		t.Error("no measures were reported by the stats collector")
	}

	for _, m := range h.Measures() {
		t.Log(m)
	}
}
