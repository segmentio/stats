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

	if pid == os.Getpid() {
		rusage := syscall.Rusage{}
		check(syscall.Getrusage(syscall.RUSAGE_SELF, &rusage))
		cpu = CPUInfo{
			User: time.Duration(rusage.Utime.Nano()),
			Sys:  time.Duration(rusage.Stime.Nano()),
		}
	} else {
		cpu = CPUInfo{
			User: clockTicksToDuration(stat.Utime),
			Sys:  clockTicksToDuration(stat.Stime),
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
