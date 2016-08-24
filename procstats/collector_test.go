package procstats

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestCollector(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient(backend)
	defer client.Close()

	c := StartCollectorWith(Config{
		CollectInterval: 100 * time.Microsecond,
		Collector: MultiCollector(
			NewGoMetrics(client),
		),
	})

	// Let the collector do a few runs.
	time.Sleep(time.Millisecond)
	c.Close()

	if len(backend.Events) == 0 {
		t.Error("no events were generated while collecting process stats")
	}
}

func TestCollectorCloser(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient(backend)
	defer client.Close()

	c := StartCollector(nil)
	c.Close()

	if len(backend.Events) != 0 {
		t.Error("unexpected events after stopping the collector")
	}
}
