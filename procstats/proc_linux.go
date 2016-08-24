package procstats

import "github.com/segmentio/stats/procstats/linux"

func collectProcMetrics(pid int) (m proc, err error) {
	code := [...](func(int, *proc) error){
		collectProcLimits,
		collectProcStat,
		collectOpenFileCount,
	}

	for i, n := 0, len(code); i != n && err == nil; i++ {
		err = code[i](pid, &m)
	}

	return
}

func collectProcLimits(pid int, m *proc) (err error) {
	var limits linux.ProcLimits

	if limits, err = linux.GetProcLimits(pid); err == nil {
		m.files.max = limits.OpenFiles.Soft
	}

	return
}

func collectProcStat(pid int, m *proc) (err error) {
	var stat linux.ProcStat

	if stat, err = linux.GetProcStat(pid); err == nil {
		// TODO: extract CPU, (see sysconf)

		m.memory.majorPageFaults = uint64(stat.Majflt)
		m.memory.minorPageFaults = uint64(stat.Minflt)

		m.threads.num = uint64(stat.NumThreads)
	}

	return
}

func collectOpenFileCount(pid int, m *proc) (err error) {
	m.files.open, err = linux.GetOpenFileCount(pid)
	return
}
