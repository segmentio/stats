package procstats

import (
	"os"

	"github.com/segmentio/stats"
)

type ProcMetrics struct {
	Pid     int
	CPU     CPU
	Memory  Memory
	IO      IO
	Files   Files
	Threads Threads
}

type CPU struct {
	// CPU time
	User stats.Counter // user cpu time used by the process
	Sys  stats.Counter // system cpu time used by the process

	// last values seen to compute the delta and increment counters
	lastUser uint64
	lastSys  uint64
}

type Memory struct {
	// Resident memory
	ResidentUsed stats.Gauge // resident memory used by the process
	ResidentMax  stats.Gauge // max resident memory the process can allocate

	// Virtual memory
	VirtualUsed stats.Gauge // virtual memory used by the process
	VirtualMax  stats.Gauge // max virtual memory the process can allocate

	// Page faults
	MajorPageFaults stats.Counter
	MinorPageFaults stats.Counter

	// last values seen to compute the delta and increment counters
	lastMajorPageFaults uint64
	lastMinorPageFaults uint64
}

type IO struct {
	// Bytes I/O.
	BytesIn  stats.Counter
	BytesOut stats.Counter

	// last values seen to compute the delta and increment counters
	lastBytesIn  uint64
	lastBytesOut uint64
}

type Files struct {
	// File descriptors
	Open stats.Gauge // fds opened by the process
	Max  stats.Gauge // max number of fds the process can open

	// Block I/O
	BlockReads  stats.Counter
	BlockWrites stats.Counter

	// last values seen to compute the delta and increment counters
	lastBlockReads  uint64
	lastBlockWrites uint64
}

type Threads struct {
	// Thread count
	Num stats.Gauge

	// Context switches
	VolountaryContextSwitches   stats.Counter
	InvolountaryContextSwitches stats.Counter

	// last values seen to compute the delta and increment counters
	lastVolountaryContextSwitches   uint64
	lastInvolountaryContextSwitches uint64
}

func NewProcMetrics(client stats.Client, tags ...stats.Tag) *ProcMetrics {
	return NewProcMetricsFor(os.Getpid(), client, tags...)
}

func NewProcMetricsFor(pid int, client stats.Client, tags ...stats.Tag) *ProcMetrics {
	return &ProcMetrics{
		Pid:     pid,
		CPU:     makeCPU(client, tags...),
		Memory:  makeMemory(client, tags...),
		IO:      makeIO(client, tags...),
		Files:   makeFiles(client, tags...),
		Threads: makeThreads(client, tags...),
	}
}

func (p *ProcMetrics) Collect() {
	if m, err := collectProcMetrics(p.Pid); err == nil {
		// CPU
		p.CPU.User.Add(float64(m.cpu.user - p.CPU.lastUser))
		p.CPU.Sys.Add(float64(m.cpu.sys - p.CPU.lastSys))

		p.CPU.lastUser = m.cpu.user
		p.CPU.lastSys = m.cpu.sys

		// Memory
		p.Memory.ResidentUsed.Set(float64(m.memory.residentUsed))
		p.Memory.ResidentMax.Set(float64(m.memory.residentMax))

		p.Memory.VirtualUsed.Set(float64(m.memory.virtualUsed))
		p.Memory.VirtualMax.Set(float64(m.memory.virtualMax))

		p.Memory.MajorPageFaults.Add(float64(m.memory.majorPageFaults - p.Memory.lastMajorPageFaults))
		p.Memory.MinorPageFaults.Add(float64(m.memory.minorPageFaults - p.Memory.lastMinorPageFaults))

		p.Memory.lastMajorPageFaults = m.memory.majorPageFaults
		p.Memory.lastMinorPageFaults = m.memory.minorPageFaults

		// IO
		p.IO.BytesIn.Add(float64(m.io.bytesIn - p.IO.lastBytesIn))
		p.IO.BytesOut.Add(float64(m.io.bytesOut - p.IO.lastBytesOut))

		p.IO.lastBytesIn = m.io.bytesIn
		p.IO.lastBytesOut = m.io.bytesOut

		// Files
		p.Files.Open.Set(float64(m.files.open))
		p.Files.Max.Set(float64(m.files.max))

		// Threads
		p.Threads.Num.Set(float64(m.threads.num))

		p.Threads.VolountaryContextSwitches.Add(float64(m.threads.volountaryContextSwitches - p.Threads.lastVolountaryContextSwitches))
		p.Threads.InvolountaryContextSwitches.Add(float64(m.threads.involountaryContextSwitches - p.Threads.lastInvolountaryContextSwitches))

		p.Threads.lastVolountaryContextSwitches = m.threads.volountaryContextSwitches
		p.Threads.lastInvolountaryContextSwitches = m.threads.involountaryContextSwitches
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
		ResidentUsed: client.Gauge("memory.resident.bytes", tags...),
		ResidentMax:  client.Gauge("memory.resident.max", tags...),

		VirtualUsed: client.Gauge("memory.virtual.bytes", tags...),
		VirtualMax:  client.Gauge("memory.virtual.max", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "major"}
	mem.MajorPageFaults = client.Counter("memory.pagefault.count", tags...)

	tags[n] = stats.Tag{"type", "minor"}
	mem.MinorPageFaults = client.Counter("memory.pagefault.count", tags...)

	return mem
}

func makeIO(client stats.Client, tags ...stats.Tag) IO {
	io := IO{}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"operation", "read"}
	io.BytesIn = client.Counter("io.bytes", tags...)

	tags[n] = stats.Tag{"operation", "write"}
	io.BytesOut = client.Counter("io.bytes", tags...)

	return io
}

func makeFiles(client stats.Client, tags ...stats.Tag) Files {
	return Files{
		Open: client.Gauge("file.open.count", tags...),
		Max:  client.Gauge("file.open.max", tags...),
	}
}

func makeThreads(client stats.Client, tags ...stats.Tag) Threads {
	threads := Threads{
		Num: client.Gauge("thread.count", tags...),
	}

	n := len(tags)
	tags = append(tags, stats.Tag{})

	tags[n] = stats.Tag{"type", "volountary"}
	threads.VolountaryContextSwitches = client.Counter("thread.switch.count", tags...)

	tags[n] = stats.Tag{"type", "involountary"}
	threads.InvolountaryContextSwitches = client.Counter("thread.switch.count", tags...)

	return threads
}

type proc struct {
	cpu     cpu
	memory  memory
	io      fio
	files   files
	threads threads
}

type cpu struct {
	user uint64
	sys  uint64
}

type memory struct {
	residentUsed    uint64
	residentMax     uint64
	virtualUsed     uint64
	virtualMax      uint64
	majorPageFaults uint64
	minorPageFaults uint64
}

type fio struct {
	bytesIn  uint64
	bytesOut uint64
}

type files struct {
	open uint64
	max  uint64
}

type threads struct {
	num                         uint64
	volountaryContextSwitches   uint64
	involountaryContextSwitches uint64
}
