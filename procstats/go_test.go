package procstats

import (
	"reflect"
	"runtime/debug"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestGoMetrics(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	gostats := NewGoMetricsWith(e)
	gostats.Collect()

	if len(h.Measures()) == 0 {
		t.Error("no measures were reported by the stats collector")
	}

	for _, m := range h.Measures() {
		t.Log(m)
	}
}

func TestStripOutdatedGCPauses(t *testing.T) {
	now := time.Now()

	gc := &debug.GCStats{}
	gc.LastGC = now.Add(-time.Second) // 1s ago
	gc.NumGC = 10                     // 10th GC pass
	gc.PauseTotal = time.Millisecond  // 1ms pauses total
	gc.Pause = []time.Duration{
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
		100 * time.Microsecond,
	}
	gc.PauseEnd = []time.Time{
		now.Add(-time.Second).Add(-100 * time.Microsecond),
		now.Add(-time.Second).Add(-200 * time.Microsecond),
		now.Add(-time.Second).Add(-300 * time.Microsecond),
		now.Add(-time.Second).Add(-400 * time.Microsecond),
		now.Add(-time.Second).Add(-500 * time.Microsecond),
		now.Add(-time.Second).Add(-600 * time.Microsecond),
		now.Add(-time.Second).Add(-700 * time.Microsecond),
		now.Add(-time.Second).Add(-800 * time.Microsecond),
		now.Add(-time.Second).Add(-900 * time.Microsecond),
		now.Add(-time.Second).Add(-1 * time.Millisecond),
	}

	gc.PauseQuantiles = nil // ???

	stripOutdatedGCPauses(gc, 8) // last observed the 8th pass

	if !reflect.DeepEqual(gc, &debug.GCStats{
		LastGC:     now.Add(-time.Second),
		NumGC:      10,
		PauseTotal: time.Millisecond,
		Pause: []time.Duration{
			100 * time.Microsecond,
			100 * time.Microsecond,
		},
		PauseEnd: []time.Time{
			now.Add(-time.Second).Add(-100 * time.Microsecond),
			now.Add(-time.Second).Add(-200 * time.Microsecond),
		},
	}) {
		t.Errorf("invalid gc stats after stripping outdated pauses:\n%#v", *gc)
	}
}
