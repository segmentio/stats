package procstats

import (
	"os"
	"syscall"
	"time"

	"github.com/segmentio/stats/procstats/linux"
)

func collectProcInfo(pid int) (info ProcInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	pagesize := uint64(syscall.Getpagesize())
	rusage := syscall.Rusage{}

	if pid == os.Getpid() {
		check(syscall.Getrusage(syscall.RUSAGE_SELF, &rusage))
	} else {
		// TODO: figure out how to get accurate stats for other processes
	}

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

	info = ProcInfo{
		CPU: CPUInfo{
			User: time.Duration(rusage.Utime.Nano()),
			Sys:  time.Duration(rusage.Stime.Nano()),
		},

		Memory: MemoryInfo{
			Available:       memoryLimit,
			Size:            pagesize * statm.Size,
			Resident:        pagesize * statm.Resident,
			Shared:          pagesize * statm.Share,
			Text:            pagesize * statm.Text,
			Data:            pagesize * statm.Data,
			MajorPageFaults: uint64(rusage.Majflt),
			MinorPageFaults: uint64(rusage.Minflt),
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
