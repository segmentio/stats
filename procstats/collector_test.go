package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestCollector(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient("test", backend)
	defer client.Close()

	collector := NewCollectorWith(CollectorConfig{
		Client:          client,
		CollectInterval: 100 * time.Microsecond,
	})

	// Let the collector do a few runs.
	time.Sleep(time.Millisecond)
	collector.Stop()

	if len(backend.Events) == 0 {
		t.Error("no events were generated while collecting process stats")
	}
}

func TestCollectorStop(t *testing.T) {
	client := stats.NewClient("test", &stats.EventBackend{})
	defer client.Close()

	collector := NewCollector(client)
	defer collector.Stop()
}
