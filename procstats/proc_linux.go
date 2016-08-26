package procstats

import (
	"syscall"
	"time"

	"github.com/segmentio/stats/procstats/linux"
)

func getpagesize() uint64 { return uint64(syscall.Getpagesize()) }

func collectProcMetrics(pid int) (m proc, err error) {
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

	m = proc{
		cpu: cpu{
			user: (time.Duration(stat.Utime) * time.Nanosecond) / time.Duration(clockTicks),
			sys:  (time.Duration(stat.Stime) * time.Nanosecond) / time.Duration(clockTicks),
		},

		memory: memory{
			available:       memoryLimit,
			size:            pagesize * statm.Size,
			resident:        pagesize * statm.Resident,
			shared:          pagesize * statm.Share,
			text:            pagesize * statm.Text,
			data:            pagesize * statm.Data,
			majorPageFaults: stat.Majflt,
			minorPageFaults: stat.Minflt,
		},

		files: files{
			open: fds,
			max:  limits.OpenFiles.Soft,
		},

		threads: threads{
			num: uint64(stat.NumThreads),
			voluntaryContextSwitches:   sched.NRVoluntarySwitches,
			involuntaryContextSwitches: sched.NRInvoluntarySwitches,
		},
	}
	return
}
