package procstats

import (
	"runtime"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestGoMetrics(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	gostats := NewGoMetricsWith(e)

	for i := 0; i != 10; i++ {
		t.Logf("collect number %d", i)
		gostats.Collect()

		if len(h.Measures()) == 0 {
			t.Error("no measures were reported by the stats collector")
		}

		for _, m := range h.Measures() {
			t.Log(m)
		}

		h.Clear()

		for j := 0; j <= i; j++ {
			runtime.GC() // to get non-zero GC stats
		}

		time.Sleep(10 * time.Millisecond)
	}
}
