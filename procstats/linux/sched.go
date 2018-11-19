package linux

import "strconv"

type ProcSched struct {
	NRSwitches            uint64 // nr_switches
	NRVoluntarySwitches   uint64 // nr_voluntary_switches
	NRInvoluntarySwitches uint64 // nr_involuntary_switches
	SEAvgLoadSum          uint64 // se.avg.load_sum
	SEAvgUtilSum          uint64 // se.avg.util_sum
	SEAvgLoadAvg          uint64 // se.avg.load_avg
	SEAvgUtilAvg          uint64 // se.avg.util_avg
}

func ReadProcSched(pid int) (proc ProcSched, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcSched(readProcFile(pid, "sched"))
	return
}

func ParseProcSched(s string) (proc ProcSched, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcSched(s)
	return
}

func parseProcSched(s string) (proc ProcSched) {
	intFields := map[string]*uint64{
		"nr_switches":             &proc.NRSwitches,
		"nr_voluntary_switches":   &proc.NRVoluntarySwitches,
		"nr_involuntary_switches": &proc.NRInvoluntarySwitches,
		"se.avg.load_sum":         &proc.SEAvgLoadSum,
		"se.avg.util_sum":         &proc.SEAvgUtilSum,
		"se.avg.load_avg":         &proc.SEAvgLoadAvg,
		"se.avg.util_avg":         &proc.SEAvgUtilAvg,
	}

	s = skipLine(s) // <progname> (<pid>, #threads: 1)
	s = skipLine(s) // -------------------------------

	forEachProperty(s, func(key string, val string) {
		if field := intFields[key]; field != nil {
			v, e := strconv.ParseUint(val, 10, 64)
			check(e)

			// There's no much doc on the structure of this file, unsure if
			// there would be a breakdown per thread... unlikely but this should
			// cover for it.
			*field += v
		}
	})

	return
}
