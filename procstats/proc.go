package procstats

import (
	"os"
	"time"

	"github.com/segmentio/stats"
)

// ProcMetrics is a metric collector that reports metrics on processes.
type ProcMetrics struct {
	engine  *stats.Engine
	pid     int
	cpu     procCPU     `metric:"cpu"`
	memory  procMemory  `metric:"memory"`
	files   procFiles   `metric:"files"`
	threads procThreads `metric:"threads"`
}

type procCPU struct {
	// CPU time
	user struct { // user cpu time used by the process
		time time.Duration `metric:"usage.seconds" type:"counter"`
		typ  string        `tag:"type"` // user
	}
	system struct { // system cpu time used by the process
		time time.Duration `metric:"usage.seconds" type:"counter"`
		typ  string        `tag:"type"` // system
	}
}

type procMemory struct {
	// Memory
	available uint64 `metric:"available.bytes" type:"gauge"` // amound of RAM available to the process
	size      uint64 `metric:"total.bytes"     type:"gauge"` // total program memory (including virtual mappings)

	resident struct { // resident set size
		usage uint64 `metric:"usage.bytes" type:"gauge"`
		typ   string `tag:"type"` // resident
	}

	shared struct { // shared pages (i.e., backed by a file)
		usage uint64 `metric:"usage.bytes" type:"gauge"`
		typ   string `tag:"type"` // shared
	}

	text struct { // text (code)
		usage uint64 `metric:"usage.bytes" type:"gauge"`
		typ   string `tag:"type"` // text
	}

	data struct { // data + stack
		usage uint64 `metric:"usage.bytes" type:"gauge"`
		typ   string `tag:"type"` // data
	}

	// Page faults
	pagefault struct {
		major struct {
			count uint64 `metric:"count" type:"counter"`
			typ   string `tag:"type"` // major
		}
		minor struct {
			count uint64 `metric:"count" type:"counter"`
			typ   string `tag:"type"` // minor
		}
	} `metric:"pagefault"`
}

type procFiles struct {
	// File descriptors
	open uint64 `metric:"open.count" type:"gauge"` // fds opened by the process
	max  uint64 `metric:"open.max"   type:"gauge"` // max number of fds the process can open
}

type procThreads struct {
	// Thread count
	num uint64 `metric:"count" type:"gauge"`

	// Context switches
	switches struct {
		voluntary struct {
			count uint64 `metric:"count"" type:"counter"`
			typ   string `tag:"type"` // voluntary
		}
		involuntary struct {
			count uint64 `metric:"count"" type:"counter"`
			typ   string `tag:"type"` // involuntary
		}
	} `metric:"switch"`
}

// NewProdMetrics collects metrics on the current process and reports them to
// the default stats engine.
func NewProcMetrics() *ProcMetrics {
	return NewProcMetricsWith(stats.DefaultEngine, os.Getpid())
}

// NewProcMetricsWith collects metrics on the process identified by pid and
// reports them to eng.
func NewProcMetricsWith(eng *stats.Engine, pid int) *ProcMetrics {
	p := &ProcMetrics{engine: eng, pid: pid}

	p.cpu.user.typ = "user"
	p.cpu.system.typ = "system"

	p.memory.resident.typ = "resident"
	p.memory.shared.typ = "shared"
	p.memory.text.typ = "text"
	p.memory.data.typ = "data"

	p.memory.pagefault.major.typ = "major"
	p.memory.pagefault.minor.typ = "minor"

	p.threads.switches.voluntary.typ = "voluntary"
	p.threads.switches.involuntary.typ = "involuntary"

	return p
}

// Collect satsifies the Collector interface.
func (p *ProcMetrics) Collect() {
	if m, err := CollectProcInfo(p.pid); err == nil {
		p.cpu.user.time = m.CPU.User
		p.cpu.system.time = m.CPU.Sys

		p.memory.available = m.Memory.Available
		p.memory.size = m.Memory.Size
		p.memory.resident.usage = m.Memory.Resident
		p.memory.shared.usage = m.Memory.Shared
		p.memory.text.usage = m.Memory.Text
		p.memory.data.usage = m.Memory.Data
		p.memory.pagefault.major.count = m.Memory.MajorPageFaults
		p.memory.pagefault.minor.count = m.Memory.MinorPageFaults

		p.files.open = m.Files.Open
		p.files.max = m.Files.Max

		p.threads.num = m.Threads.Num
		p.threads.switches.voluntary.count = m.Threads.VoluntaryContextSwitches
		p.threads.switches.involuntary.count = m.Threads.InvoluntaryContextSwitches

		p.engine.Report(p)
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
