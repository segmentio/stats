package procstats

import (
	"runtime"

	"github.com/segmentio/stats"
)

type RuntimeStats struct {
	NumCPU       stats.Gauge
	NumCgoCall   stats.Gauge
	NumGoroutine stats.Gauge
}

func NewRuntimeStats(client stats.Client, tags ...stats.Tag) *RuntimeStats {
	tags = append(tags, stats.Tag{"go_version", runtime.Version()})
	return &RuntimeStats{
		NumCPU:       client.Gauge("runtime.goroutine.count", tags...),
		NumCgoCall:   client.Gauge("runtime.cgo_call.count", tags...),
		NumGoroutine: client.Gauge("runtime.cpu.count", tags...),
	}
}

func (rt *RuntimeStats) Collect() {
	rt.NumCPU.Set(float64(runtime.NumCPU()))
	rt.NumCgoCall.Set(float64(runtime.NumCgoCall()))
	rt.NumGoroutine.Set(float64(runtime.NumGoroutine()))
}
