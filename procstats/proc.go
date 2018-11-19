package procstats

import (
	"os"
	"runtime"
	"time"

	"github.com/segmentio/stats"
)

// ProcMetrics is a metric collector that reports metrics on processes.
type ProcMetrics struct {
	engine   *stats.Engine
	pid      int
	cpu      procCPU     `metric:"cpu"`
	memory   procMemory  `metric:"memory"`
	files    procFiles   `metric:"files"`
	threads  procThreads `metric:"threads"`
	last     ProcInfo
	lastTime time.Time
}

type procCPU struct {
	// CPU time
	user struct { // user cpu time used by the process
		time    time.Duration `metric:"usage.seconds" type:"counter"`
		percent float64       `metric:"usage.percent" type:"gauge"`
		typ     string        `tag:"type"` // user
	}
	system struct { // system cpu time used by the process
		time    time.Duration `metric:"usage.seconds" type:"counter"`
		percent float64       `metric:"usage.percent" type:"gauge"`
		typ     string        `tag:"type"` // system
	}
	total struct {
		time    time.Duration `metric:"usage_total.seconds" type:"counter"`
		percent float64       `metric:"usage_total.percent" type:"gauge"`
	}
}

type procMemory struct {
	// Memory
	available uint64 `metric:"available.bytes" type:"gauge"` // amound of RAM available to the process
	size      uint64 `metric:"total.bytes"     type:"gauge"` // total program memory (including virtual mappings)

	resident struct { // resident set size
		usage   uint64  `metric:"usage.bytes"   type:"gauge"`
		percent float64 `metric:"usage.percent" type:"gauge"`
		typ     string  `tag:"type"` // resident
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
			count uint64 `metric:"count" type:"counter"`
			typ   string `tag:"type"` // voluntary
		}
		involuntary struct {
			count uint64 `metric:"count" type:"counter"`
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

// Collect satisfies the Collector interface.
func (p *ProcMetrics) Collect() {
	if m, err := CollectProcInfo(p.pid); err == nil {
		now := time.Now()

		if !p.lastTime.IsZero() {
			ratio := 1.0
			switch {
			case m.CPU.Period > 0 && m.CPU.Quota > 0:
				ratio = float64(m.CPU.Quota) / float64(m.CPU.Period)
			case m.CPU.Shares > 0:
				ratio = float64(m.CPU.Shares) / 1024
			default:
				ratio = 1 / float64(runtime.NumCPU())
			}

			interval := ratio * float64(now.Sub(p.lastTime))

			p.cpu.user.time = m.CPU.User - p.last.CPU.User
			p.cpu.user.percent = 100 * float64(p.cpu.user.time) / interval

			p.cpu.system.time = m.CPU.Sys - p.last.CPU.Sys
			p.cpu.system.percent = 100 * float64(p.cpu.system.time) / interval

			p.cpu.total.time = (m.CPU.User + m.CPU.Sys) - (p.last.CPU.User + p.last.CPU.Sys)
			p.cpu.total.percent = 100 * float64(p.cpu.total.time) / interval

		}

		p.memory.available = m.Memory.Available
		p.memory.size = m.Memory.Size
		p.memory.resident.usage = m.Memory.Resident
		p.memory.resident.percent = 100 * float64(p.memory.resident.usage) / float64(p.memory.available)
		p.memory.shared.usage = m.Memory.Shared
		p.memory.text.usage = m.Memory.Text
		p.memory.data.usage = m.Memory.Data
		p.memory.pagefault.major.count = m.Memory.MajorPageFaults - p.last.Memory.MajorPageFaults
		p.memory.pagefault.minor.count = m.Memory.MinorPageFaults - p.last.Memory.MinorPageFaults

		p.files.open = m.Files.Open
		p.files.max = m.Files.Max

		p.threads.num = m.Threads.Num
		p.threads.switches.voluntary.count = m.Threads.VoluntaryContextSwitches - p.last.Threads.VoluntaryContextSwitches
		p.threads.switches.involuntary.count = m.Threads.InvoluntaryContextSwitches - p.last.Threads.InvoluntaryContextSwitches

		p.last = m
		p.lastTime = now
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

	// Linux-specific details about the CPU configuration of the process.
	//
	// The values are all zero if they are not known.
	//
	// For more details on what those values represent see:
	//	https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt
	//	https://kernel.googlesource.com/pub/scm/linux/kernel/git/glommer/memcg/+/cpu_stat/Documentation/cgroups/cpu.txt
	Period time.Duration // scheduler period
	Quota  time.Duration // time quota in the scheduler period
	Shares int64         // 1024 scaled value representing the CPU shares
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

type FileInfo struct {
	Open uint64 // fds opened by the process
	Max  uint64 // max number of fds the process can open
}

type ThreadInfo struct {
	Num                        uint64
	VoluntaryContextSwitches   uint64
	InvoluntaryContextSwitches uint64
}
