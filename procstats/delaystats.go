package procstats

import (
	"os"
	"time"

	stats "github.com/segmentio/stats/v5"
)

// DelayMetrics is a metric collector that reports resource delays on processes.
type DelayMetrics struct {
	engine *stats.Engine
	pid    int

	CPUDelay       time.Duration `metric:"cpu.delay.seconds" type:"counter"`
	BlockIODelay   time.Duration `metric:"blockio.delay.seconds" type:"counter"`
	SwapInDelay    time.Duration `metric:"swapin.delay.seconds" type:"counter"`
	FreePagesDelay time.Duration `metric:"freepages.delay.seconds" type:"counter"`
}

// NewDelayMetrics collects metrics on the current process and reports them to
// the default stats engine.
func NewDelayMetrics() *DelayMetrics {
	return NewDelayMetricsWith(stats.DefaultEngine, os.Getpid())
}

// NewDelayMetricsWith collects metrics on the process identified by pid and
// reports them to eng.
func NewDelayMetricsWith(eng *stats.Engine, pid int) *DelayMetrics {
	return &DelayMetrics{engine: eng, pid: pid}
}

// Collect satisfies the Collector interface.
func (d *DelayMetrics) Collect() {
	if info, err := CollectDelayInfo(d.pid); err == nil {
		d.CPUDelay = info.CPUDelay
		d.BlockIODelay = info.BlockIODelay
		d.SwapInDelay = info.SwapInDelay
		d.FreePagesDelay = info.FreePagesDelay
		d.engine.Report(d)
	}
}

// DelayInfo stores delay Durations for various resources.
type DelayInfo struct {
	CPUDelay       time.Duration
	BlockIODelay   time.Duration
	SwapInDelay    time.Duration
	FreePagesDelay time.Duration
}

// CollectDelayInfo returns DelayInfo for a pid and an error, if any.
func CollectDelayInfo(pid int) (DelayInfo, error) {
	return collectDelayInfo(pid)
}
