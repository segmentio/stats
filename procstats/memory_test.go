package procstats

import (
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestMemoryStats(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient("test", backend)
	defer client.Close()

	memstats := NewMemoryStats(client)
	memstats.Collect()

	if len(backend.Events) == 0 {
		t.Error("no events were generated while collecting memory stats")
	}

	for i, e := range backend.Events {
		if !strings.HasPrefix(e.Name, "test.memory.") {
			t.Errorf("invalid event name for event #%d: %s", i, e.Name)
		}
		t.Logf("- %v", e)
	}
}

func TestMemoryStatsMock(t *testing.T) {
	now := time.Now()

	backend := &stats.EventBackend{}
	client := stats.NewClient("test", backend)
	defer client.Close()

	memstats := NewMemoryStats(client)
	memstats.gc.NumGC = 1
	memstats.gc.Pause = []time.Duration{time.Microsecond}
	memstats.gc.PauseEnd = []time.Time{now.Add(-time.Second)}
	memstats.report(time.Now(), time.Microsecond, time.Microsecond)

	if len(backend.Events) == 0 {
		t.Error("no events were generated while collecting memory stats")
	}

	for i, e := range backend.Events {
		if !strings.HasPrefix(e.Name, "test.memory.") {
			t.Errorf("invalid event name for event #%d: %s", i, e.Name)
		}
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
