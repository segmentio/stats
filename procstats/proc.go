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
	if m, err := CollectProcInfo(p.pid); err == nil {
		// CPU
		p.cpu.user.Set(m.CPU.User.Seconds())
		p.cpu.sys.Set(m.CPU.Sys.Seconds())

		// Memory
		p.memory.available.Set(float64(m.Memory.Available))
		p.memory.size.Set(float64(m.Memory.Size))
		p.memory.resident.Set(float64(m.Memory.Resident))
		p.memory.shared.Set(float64(m.Memory.Shared))
		p.memory.text.Set(float64(m.Memory.Text))
		p.memory.data.Set(float64(m.Memory.Data))
		p.memory.majorPageFaults.Set(float64(m.Memory.MajorPageFaults))
		p.memory.minorPageFaults.Set(float64(m.Memory.MinorPageFaults))

		// Files
		p.files.open.Set(float64(m.Files.Open))
		p.files.max.Set(float64(m.Files.Max))

		// Threads
		p.threads.num.Set(float64(m.Threads.Num))
		p.threads.voluntaryContextSwitches.Set(float64(m.Threads.VoluntaryContextSwitches))
		p.threads.involuntaryContextSwitches.Set(float64(m.Threads.InvoluntaryContextSwitches))
	}
}

type ProcInfo struct {
	CPU     CPUInfo
	Memory  MemoryInfo
	Files   FileInfo
	Threads ThreadInfo
}

func CollectProcInfo(pid int) (ProcInfo, error) {
	return collectProcInfo(pid)
}

type CPUInfo struct {
	User time.Duration // user cpu time used by the process
	Sys  time.Duration // system cpu time used by the process
}

func CollectCPUInfo(pid int) (CPUInfo, error) {
	return collectCPUInfo(pid)
}

type MemoryInfo struct {
	Available uint64 // amound of RAM available to the process
	Size      uint64 // total program memory (including virtual mappings)
	Resident  uint64 // resident set size
	Shared    uint64 // shared pages (i.e., backed by a file)
	Text      uint64 // text (code)
	Data      uint64 // data + stack

	MajorPageFaults uint64
	MinorPageFaults uint64
}

func CollectMemoryInfo(pid int) (MemoryInfo, error) {
	return collectMemoryInfo(pid)
}

type FileInfo struct {
	Open uint64 // fds opened by the process
	Max  uint64 // max number of fds the process can open
}

func CollectFileInfo(pid int) (FileInfo, error) {
	return collectFileInfo(pid)
}

type ThreadInfo struct {
	Num                        uint64
	VoluntaryContextSwitches   uint64
	InvoluntaryContextSwitches uint64
}

func CollectThreadInfo(pid int) (ThreadInfo, error) {
	return collectThreadInfo(pid)
}
