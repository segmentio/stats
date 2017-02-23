package procstats

import (
	"os"
	"time"

	"github.com/segmentio/stats"
)

// ProcMetrics is a metric collector that reports metrics on processes.
type ProcMetrics struct {
	pid     int
	cpu     procCPU
	memory  procMemory
	files   procFiles
	threads procThreads
}

type procCPU struct {
	// CPU time
	user stats.Counter // user cpu time used by the process
	sys  stats.Counter // system cpu time used by the process
}

type procMemory struct {
	// Memory
	available stats.Gauge // amound of RAM available to the process
	size      stats.Gauge // total program memory (including virtual mappings)
	resident  stats.Gauge // resident set size
	shared    stats.Gauge // shared pages (i.e., backed by a file)
	text      stats.Gauge // text (code)
	data      stats.Gauge // data + stack

	// Page faults
	majorPageFaults stats.Counter
	minorPageFaults stats.Counter
}

type procFiles struct {
	// File descriptors
	open stats.Gauge // fds opened by the process
	max  stats.Gauge // max number of fds the process can open
}

type procThreads struct {
	// Thread count
	num stats.Gauge

	// Context switches
	voluntaryContextSwitches   stats.Counter
	involuntaryContextSwitches stats.Counter
}

// NewProdMetrics collects metrics on the current process and reports them to
// the default stats engine.
func NewProcMetrics() *ProcMetrics {
	return NewProcMetricsWith(stats.DefaultEngine, os.Getpid())
}

// NewProcMetricsWith collects metrics on the process identified by pid and
// reports them to eng.
func NewProcMetricsWith(eng *stats.Engine, pid int) *ProcMetrics {
	return &ProcMetrics{
		pid: pid,
		cpu: procCPU{
			user: *eng.Counter("cpu.usage.seconds", stats.Tag{"type", "user"}),
			sys:  *eng.Counter("cpu.usage.seconds", stats.Tag{"type", "system"}),
		},
		memory: procMemory{
			available:       *eng.Gauge("memory.available.bytes"),
			size:            *eng.Gauge("memory.total.bytes"),
			resident:        *eng.Gauge("memory.usage.bytes", stats.Tag{"type", "resident"}),
			shared:          *eng.Gauge("memory.usage.bytes", stats.Tag{"type", "shared"}),
			text:            *eng.Gauge("memory.usage.bytes", stats.Tag{"type", "text"}),
			data:            *eng.Gauge("memory.usage.bytes", stats.Tag{"type", "data"}),
			majorPageFaults: *eng.Counter("memory.pagefault.count", stats.Tag{"type", "major"}),
			minorPageFaults: *eng.Counter("memory.pagefault.count", stats.Tag{"type", "minor"}),
		},
		files: procFiles{
			open: *eng.Gauge("files.open.count"),
			max:  *eng.Gauge("files.open.max"),
		},
		threads: procThreads{
			num: *eng.Gauge("thread.count"),
			voluntaryContextSwitches:   *eng.Counter("thread.switch.count", stats.Tag{"type", "voluntary"}),
			involuntaryContextSwitches: *eng.Counter("thread.switch.count", stats.Tag{"type", "involuntary"}),
		},
	}
}

// Collect satsifies the Collector interface.
func (p *ProcMetrics) Collect() {
	if m, err := collectProcMetrics(p.pid); err == nil {
		// CPU
		p.cpu.user.Set(m.cpu.user.Seconds())
		p.cpu.sys.Set(m.cpu.sys.Seconds())

		// Memory
		p.memory.available.Set(float64(m.memory.available))
		p.memory.size.Set(float64(m.memory.size))
		p.memory.resident.Set(float64(m.memory.resident))
		p.memory.shared.Set(float64(m.memory.shared))
		p.memory.text.Set(float64(m.memory.text))
		p.memory.data.Set(float64(m.memory.data))
		p.memory.majorPageFaults.Set(float64(m.memory.majorPageFaults))
		p.memory.minorPageFaults.Set(float64(m.memory.minorPageFaults))

		// Files
		p.files.open.Set(float64(m.files.open))
		p.files.max.Set(float64(m.files.max))

		// Threads
		p.threads.num.Set(float64(m.threads.num))
		p.threads.voluntaryContextSwitches.Set(float64(m.threads.voluntaryContextSwitches))
		p.threads.involuntaryContextSwitches.Set(float64(m.threads.involuntaryContextSwitches))
	}
}

// =============================================================================
// These structs are used by the platform-specific implementations as a bridge
// to the high-level API.

type proc struct {
	cpu
	memory
	files
	threads
}

type cpu struct {
	user time.Duration
	sys  time.Duration
}

type memory struct {
	available uint64
	size      uint64
	resident  uint64
	shared    uint64
	text      uint64
	data      uint64

	majorPageFaults uint64
	minorPageFaults uint64
}

type files struct {
	open uint64
	max  uint64
}

type threads struct {
	num                        uint64
	voluntaryContextSwitches   uint64
	involuntaryContextSwitches uint64
}

// =============================================================================
