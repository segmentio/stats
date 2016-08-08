package procstats

import (
	"runtime"

	"github.com/segmentio/stats"
)

type RuntimeStats struct {
	NumCPU       stats.Gauge
	NumCgoCall   stats.Gauge
	NumGoroutine stats.Gauge
	NumThread    stats.Gauge
	NumFD        stats.Gauge
	NumFDMax     stats.Gauge
}

func NewRuntimeStats(client stats.Client, tags ...stats.Tag) *RuntimeStats {
	return &RuntimeStats{
		NumCPU:       client.Gauge("runtime.cpu.count", tags...),
		NumCgoCall:   client.Gauge("runtime.cgo_call.count", tags...),
		NumGoroutine: client.Gauge("runtime.goroutine.count", tags...),
		NumThread:    client.Gauge("runtime.thread.count", tags...),
		NumFD:        client.Gauge("runtime.fd.count", tags...),
		NumFDMax:     client.Gauge("runtime.fd_max.count", tags...),
	}
}

func (rt *RuntimeStats) Collect() {
	rt.NumCPU.Set(float64(runtime.NumCPU()))
	rt.NumCgoCall.Set(float64(runtime.NumCgoCall()))
	rt.NumGoroutine.Set(float64(runtime.NumGoroutine()))
	rt.NumThread.Set(float64(threadCount()))
	rt.NumFD.Set(float64(fdCount()))
	rt.NumFDMax.Set(float64(fdMax()))
}
