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

	// last values seen to compute the delta and increment counters
	lastUser time.Duration
	lastSys  time.Duration
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

	// last values seen to compute the delta and increment counters
	lastMajorPageFaults uint64
	lastMinorPageFaults uint64
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

	// last values seen to compute the delta and increment counters
	lastVoluntaryContextSwitches   uint64
	lastInvoluntaryContextSwitches uint64
}

func NewProcMetrics(client stats.Client, tags ...stats.Tag) *ProcMetrics {
	return NewProcMetricsFor(os.Getpid(), client, tags...)
}

func NewProcMetricsFor(pid int, client stats.Client, tags ...stats.Tag) *ProcMetrics {
	return &ProcMetrics{
		Pid:     pid,
		CPU:     makeCPU(client, tags...),
		Memory:  makeMemory(client, tags...),
		Files:   makeFiles(client, tags...),
		Threads: makeThreads(client, tags...),
	}
}

func (p *ProcMetrics) Collect() {
	if m, err := collectProcMetrics(p.Pid); err == nil {
		// CPU
		p.CPU.User.Add((m.cpu.user - p.CPU.lastUser).Seconds())
		p.CPU.Sys.Add((m.cpu.sys - p.CPU.lastSys).Seconds())

		p.CPU.lastUser = m.cpu.user
		p.CPU.lastSys = m.cpu.sys

		// Memory
		p.Memory.Available.Set(float64(m.memory.available))
		p.Memory.Size.Set(float64(m.memory.size))
		p.Memory.Resident.Set(float64(m.memory.resident))
		p.Memory.Shared.Set(float64(m.memory.shared))
		p.Memory.Text.Set(float64(m.memory.text))
		p.Memory.Data.Set(float64(m.memory.data))

		p.Memory.MajorPageFaults.Add(float64(m.memory.majorPageFaults - p.Memory.lastMajorPageFaults))
		p.Memory.MinorPageFaults.Add(float64(m.memory.minorPageFaults - p.Memory.lastMinorPageFaults))

		p.Memory.lastMajorPageFaults = m.memory.majorPageFaults
		p.Memory.lastMinorPageFaults = m.memory.minorPageFaults

		// Files
		p.Files.Open.Set(float64(m.files.open))
		p.Files.Max.Set(float64(m.files.max))

		// Threads
		p.Threads.Num.Set(float64(m.threads.num))

		p.Threads.VoluntaryContextSwitches.Add(float64(m.threads.voluntaryContextSwitches - p.Threads.lastVoluntaryContextSwitches))
		p.Threads.InvoluntaryContextSwitches.Add(float64(m.threads.involuntaryContextSwitches - p.Threads.lastInvoluntaryContextSwitches))

		p.Threads.lastVoluntaryContextSwitches = m.threads.voluntaryContextSwitches
		p.Threads.lastInvoluntaryContextSwitches = m.threads.involuntaryContextSwitches
	}
}

func makeCPU(client stats.Client, tags ...stats.Tag) (cpu CPU) {
	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "user"}
	cpu.User = client.Counter("cpu.usage.seconds", tags...)

	tags[n] = stats.Tag{"type", "sys"}
	cpu.Sys = client.Counter("cpu.usage.seconds", tags...)

	return cpu
}

func makeMemory(client stats.Client, tags ...stats.Tag) Memory {
	mem := Memory{
		Available: client.Gauge("memory.available.bytes", tags...),
		Size:      client.Gauge("memory.total.bytes", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "resident"}
	mem.Resident = client.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "shared"}
	mem.Shared = client.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "text"}
	mem.Text = client.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "data"}
	mem.Data = client.Gauge("memory.usage.bytes", tags...)

	tags[n] = stats.Tag{"type", "major"}
	mem.MajorPageFaults = client.Counter("memory.pagefault.count", tags...)

	tags[n] = stats.Tag{"type", "minor"}
	mem.MinorPageFaults = client.Counter("memory.pagefault.count", tags...)

	return mem
}

func makeFiles(client stats.Client, tags ...stats.Tag) Files {
	return Files{
		Open: client.Gauge("files.open.count", tags...),
		Max:  client.Gauge("files.open.max", tags...),
	}
}

func makeThreads(client stats.Client, tags ...stats.Tag) Threads {
	threads := Threads{
		Num: client.Gauge("thread.count", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "voluntary"}
	threads.VoluntaryContextSwitches = client.Counter("thread.switch.count", tags...)

	tags[n] = stats.Tag{"type", "involuntary"}
	threads.InvoluntaryContextSwitches = client.Counter("thread.switch.count", tags...)

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
