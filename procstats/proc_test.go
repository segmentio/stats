package procstats

import (
	"testing"

	"github.com/segmentio/stats"
)

func TestProcMetrics(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient(backend)
	defer client.Close()

	proc := NewProcMetrics(client)
	proc.Collect()

	if len(backend.Events) == 0 {
		t.Errorf("no metrics reported by process metric collector")
	}
}
