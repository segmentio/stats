package procstats

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/stats/procstats/linux"
)

var (
	clockTickOnce  sync.Once
	clockTickHertz uint64
	clockTickError error
)

func clockTicksToDuration(ticks uint64) time.Duration {
	clockTickOnce.Do(func() { clockTickHertz, clockTickError = clockTick() })
	check(clockTickError)
	return time.Duration(1e9 * float64(ticks) / float64(clockTickHertz))
}

func clockTick() (uint64, error) {
	s, err := getconf("CLK_TCK")
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(s), 10, 64)
}

func getconf(name string) (string, error) {
	file, err := exec.LookPath("getconf")
	if err != nil {
		return "", err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return "", nil
	}
	defer r.Close()
	defer w.Close()

	p, err := os.StartProcess(file, []string{"getconf", name}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, w, os.Stderr},
	})
	if err != nil {
		return "", err
	}

	w.Close()
	b, err := ioutil.ReadAll(r)
	p.Wait()
	return string(b), err
}

func collectProcInfo(pid int) (info ProcInfo, err error) {
	defer func() { err = convertPanicToError(recover()) }()

	var pagesize = uint64(syscall.Getpagesize())
	var cpu CPUInfo

	memoryLimit, err := linux.ReadMemoryLimit(pid)
	check(err)

	limits, err := linux.ReadProcLimits(pid)
	check(err)

	stat, err := linux.ReadProcStat(pid)
	check(err)

	statm, err := linux.ReadProcStatm(pid)
	check(err)

	sched, err := linux.ReadProcSched(pid)
	check(err)

	fds, err := linux.ReadOpenFileCount(pid)
	check(err)

	if pid == os.Getpid() {
		rusage := syscall.Rusage{}
		check(syscall.Getrusage(syscall.RUSAGE_SELF, &rusage))

		cpuPeriod, _ := linux.ReadCPUPeriod("")
		cpuQuota, _ := linux.ReadCPUQuota("")
		cpuShares, _ := linux.ReadCPUShares("")

		cpu = CPUInfo{
			User:   time.Duration(rusage.Utime.Nano()),
			Sys:    time.Duration(rusage.Stime.Nano()),
			Period: cpuPeriod,
			Quota:  cpuQuota,
			Shares: cpuShares,
		}
	} else {
		selfCGroups, _ := linux.ReadProcCGroup(os.Getpid())
		procCGroups, _ := linux.ReadProcCGroup(pid)

		selfCPU, _ := selfCGroups.Lookup("cpu,cpuacct")
		procCPU, _ := procCGroups.Lookup("cpu,cpuacct")

		var (
			cpuPeriod time.Duration
			cpuQuota  time.Duration
			cpuShares int64
		)

		// The calling program and the target are both in the same cgroup, we
		// can use the cpu period, quota, and shares of the calling process.
		//
		// We do this instead of looking directly into the cgroup directory
		// because we don't have any garantees that the path is exposed to the
		// current process (but it should always have access to its own cgroup).
		if selfCPU.Path == procCPU.Path {
			cpuPeriod, _ = linux.ReadCPUPeriod("")
			cpuQuota, _ = linux.ReadCPUQuota("")
			cpuShares, _ = linux.ReadCPUShares("")
		} else {
			cpuPeriod, _ = linux.ReadCPUPeriod(procCPU.Path)
			cpuQuota, _ = linux.ReadCPUQuota(procCPU.Path)
			cpuShares, _ = linux.ReadCPUShares(procCPU.Path)
		}

		cpu = CPUInfo{
			User:   clockTicksToDuration(stat.Utime),
			Sys:    clockTicksToDuration(stat.Stime),
			Period: cpuPeriod,
			Quota:  cpuQuota,
			Shares: cpuShares,
		}
	}

	info = ProcInfo{
		CPU: cpu,

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
