package procstats

import (
	"strings"
	"testing"

	"github.com/segmentio/stats"
)

func TestRusageStats(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient("test", backend)
	defer client.Close()

	rtstats := NewRusageStats(client)
	rtstats.Collect()

	if len(backend.Events) == 0 {
		t.Error("no events were generated while collecting rusage stats")
	}

	for i, e := range backend.Events {
		if !strings.HasPrefix(e.Name, "test.rusage.") {
			t.Errorf("invalid event name for event #%d: %s", i, e.Name)
		}
		t.Logf("- %v", e)
	}
}
