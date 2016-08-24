package procstats

import "github.com/segmentio/stats/procstats/linux"

func collectProcMetrics(pid int) (m proc, err error) {
	code := [...](func(int, *proc) error){
		collectProcLimits,
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

func collectOpenFileCount(pid int, m *proc) (err error) {
	m.files.open, err = linux.GetOpenFileCount(pid)
	return
}
