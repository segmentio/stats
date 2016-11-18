package procstats

import (
	"os"
	"time"

	"github.com/segmentio/stats"
)

type ProcMetrics struct {
	Pid     int
	CPU     CPU
	Memory  Memory
	Files   Files
	Threads Threads
}

type CPU struct {
	// CPU time
	User stats.Counter // user cpu time used by the process
	Sys  stats.Counter // system cpu time used by the process
}

type Memory struct {
	// Memory
	Available stats.Gauge // amound of RAM available to the process
	Size      stats.Gauge // total program memory (including virtual mappings)
	Resident  stats.Gauge // resident set size
	Shared    stats.Gauge // shared pages (i.e., backed by a file)
	Text      stats.Gauge // text (code)
	Data      stats.Gauge // data + stack

	// Page faults
	MajorPageFaults stats.Counter
	MinorPageFaults stats.Counter
}

type Files struct {
	// File descriptors
	Open stats.Gauge // fds opened by the process
	Max  stats.Gauge // max number of fds the process can open
}

type Threads struct {
	// Thread count
	Num stats.Gauge

	// Context switches
	VoluntaryContextSwitches   stats.Counter
	InvoluntaryContextSwitches stats.Counter
}

func NewProcMetrics(eng *stats.Engine, tags ...stats.Tag) *ProcMetrics {
	return NewProcMetricsFor(os.Getpid(), eng, tags...)
}

func NewProcMetricsFor(pid int, eng *stats.Engine, tags ...stats.Tag) *ProcMetrics {
	return &ProcMetrics{
		Pid:     pid,
		CPU:     makeCPU(eng, tags...),
		Memory:  makeMemory(eng, tags...),
		Files:   makeFiles(eng, tags...),
		Threads: makeThreads(eng, tags...),
	}
}

func (p *ProcMetrics) Collect() {
	if m, err := collectProcMetrics(p.Pid); err == nil {
		// CPU
		p.CPU.User.Set(m.cpu.user.Seconds())
		p.CPU.Sys.Set(m.cpu.sys.Seconds())

		// Memory
		p.Memory.Available.Set(float64(m.memory.available))
		p.Memory.Size.Set(float64(m.memory.size))
		p.Memory.Resident.Set(float64(m.memory.resident))
		p.Memory.Shared.Set(float64(m.memory.shared))
		p.Memory.Text.Set(float64(m.memory.text))
		p.Memory.Data.Set(float64(m.memory.data))
		p.Memory.MajorPageFaults.Set(float64(m.memory.majorPageFaults))
		p.Memory.MinorPageFaults.Set(float64(m.memory.minorPageFaults))

		// Files
		p.Files.Open.Set(float64(m.files.open))
		p.Files.Max.Set(float64(m.files.max))

		// Threads
		p.Threads.Num.Set(float64(m.threads.num))
		p.Threads.VoluntaryContextSwitches.Set(float64(m.threads.voluntaryContextSwitches))
		p.Threads.InvoluntaryContextSwitches.Set(float64(m.threads.involuntaryContextSwitches))
	}
}

func makeCPU(eng *stats.Engine, tags ...stats.Tag) (cpu CPU) {
	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "user"}
	cpu.User = eng.Counter("cpu.usage.seconds", tags...)

	tags[n] = stats.Tag{"type", "sys"}
	cpu.Sys = eng.Counter("cpu.usage.seconds", tags...)

	return cpu
}

func makeMemory(eng *stats.Engine, tags ...stats.Tag) Memory {
	mem := Memory{
		Available: eng.Gauge("memory.available.bytes", tags...),
		Size:      eng.Gauge("memory.total.bytes", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "resident"}
	mem.Resident = eng.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "shared"}
	mem.Shared = eng.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "text"}
	mem.Text = eng.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "data"}
	mem.Data = eng.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "major"}
	mem.MajorPageFaults = eng.Counter("memory.pagefault.count", tags...)

	tags[n] = stats.Tag{"type", "minor"}
	mem.MinorPageFaults = eng.Counter("memory.pagefault.count", tags...)

	return mem
}

func makeFiles(eng *stats.Engine, tags ...stats.Tag) Files {
	return Files{
		Open: eng.Gauge("files.open.count", tags...),
		Max:  eng.Gauge("files.open.max", tags...),
	}
}

func makeThreads(eng *stats.Engine, tags ...stats.Tag) Threads {
	threads := Threads{
		Num: eng.Gauge("thread.count", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "voluntary"}
	threads.VoluntaryContextSwitches = eng.Counter("thread.switch.count", tags...)

	tags[n] = stats.Tag{"type", "involuntary"}
	threads.InvoluntaryContextSwitches = eng.Counter("thread.switch.count", tags...)

	return threads
}

type proc struct {
	cpu     cpu
	memory  memory
	files   files
	threads threads
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
