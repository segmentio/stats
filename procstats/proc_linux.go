package procstats

import (
	"syscall"
	"time"

	"github.com/segmentio/stats/procstats/linux"
)

func getpagesize() uint64 { return uint64(syscall.Getpagesize()) }

func collectProcInfo(pid int) (info ProcInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	memoryLimit, err := linux.GetMemoryLimit(pid)
	check(err)

	limits, err := linux.GetProcLimits(pid)
	check(err)

	stat, err := linux.GetProcStat(pid)
	check(err)

	statm, err := linux.GetProcStatm(pid)
	check(err)

	sched, err := linux.GetProcSched(pid)
	check(err)

	fds, err := linux.GetOpenFileCount(pid)
	check(err)

	pagesize := getpagesize()
	clockTicks := getclktck()

	info = ProcInfo{
		CPU: CPUInfo{
			User: (time.Duration(stat.Utime) * time.Nanosecond) / time.Duration(clockTicks),
			Sys:  (time.Duration(stat.Stime) * time.Nanosecond) / time.Duration(clockTicks),
		},

		Memory: MemoryInfo{
			Available:       memoryLimit,
			Size:            pagesize * statm.Size,
			Resident:        pagesize * statm.Resident,
			Shared:          pagesize * statm.Share,
			Text:            pagesize * statm.Text,
			Data:            pagesize * statm.Data,
			MajorPageFaults: stat.Majflt,
			MinorPageFaults: stat.Minflt,
		},

		Files: FileInfo{
			Open: fds,
			Max:  limits.OpenFiles.Soft,
		},

		Threads: ThreadInfo{
			Num: uint64(stat.NumThreads),
			VoluntaryContextSwitches:   sched.NRVoluntarySwitches,
			InvoluntaryContextSwitches: sched.NRInvoluntarySwitches,
		},
	}
	return
}

func collectCPUInfo(pid int) (info CPUInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	stat, err := linux.GetProcStat(pid)
	check(err)

	clockTicks := getclktck()

	info = CPUInfo{
		User: (time.Duration(stat.Utime) * time.Nanosecond) / time.Duration(clockTicks),
		Sys:  (time.Duration(stat.Stime) * time.Nanosecond) / time.Duration(clockTicks),
	}
	return
}

func collectMemoryInfo(pid int) (info MemoryInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	memoryLimit, err := linux.GetMemoryLimit(pid)
	check(err)

	stat, err := linux.GetProcStat(pid)
	check(err)

	statm, err := linux.GetProcStatm(pid)
	check(err)

	pagesize := getpagesize()

	info = MemoryInfo{
		Available:       memoryLimit,
		Size:            pagesize * statm.Size,
		Resident:        pagesize * statm.Resident,
		Shared:          pagesize * statm.Share,
		Text:            pagesize * statm.Text,
		Data:            pagesize * statm.Data,
		MajorPageFaults: stat.Majflt,
		MinorPageFaults: stat.Minflt,
	}
	return
}

func collectFileInfo(pid int) (info FileInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	limits, err := linux.GetProcLimits(pid)
	check(err)

	fds, err := linux.GetOpenFileCount(pid)
	check(err)

	info = FileInfo{
		Open: fds,
		Max:  limits.OpenFiles.Soft,
	}
	return
}

func collectThreadInfo(pid int) (info ThreadInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	stat, err := linux.GetProcStat(pid)
	check(err)

	sched, err := linux.GetProcSched(pid)
	check(err)

	info = ThreadInfo{
		Num: uint64(stat.NumThreads),
		VoluntaryContextSwitches:   sched.NRVoluntarySwitches,
		InvoluntaryContextSwitches: sched.NRInvoluntarySwitches,
	}
	return
}
