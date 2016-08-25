package procstats

// #include <unistd.h>
import "C"
import (
	"syscall"
	"time"

	"github.com/segmentio/stats/procstats/linux"
)

func collectProcMetrics(pid int) (m proc, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	clockTicks := uint64(C.sysconf(C._SC_CLK_TCK))
	pagesize := uint64(syscall.Getpagesize())

	limits, err := linux.GetProcLimits(pid)
	check(err)

	stat, err := linux.GetProcStat(pid)
	check(err)

	sysinfo := syscall.Sysinfo_t{}
	check(syscall.Sysinfo(&sysinfo))

	statm, err := linux.GetProcStatm(pid)
	check(err)

	sched, err := linux.GetProcSched(pid)
	check(err)

	fds, err := linux.GetOpenFileCount(pid)
	check(err)

	m = proc{
		cpu: cpu{
			user: (time.Duration(stat.Utime) * time.Nanosecond) / time.Duration(clockTicks),
			sys:  (time.Duration(stat.Stime) * time.Nanosecond) / time.Duration(clockTicks),
		},

		memory: memory{
			available:       uint64(sysinfo.Unit) * sysinfo.Totalram,
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
