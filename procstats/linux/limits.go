package linux

import "strconv"

const (
	Unlimited uint64 = 1<<64 - 1
)

type Limits struct {
	Name string
	Soft uint64
	Hard uint64
	Unit string
}

type ProcLimits struct {
	CPUTime          Limits // seconds
	FileSize         Limits // bytes
	DataSize         Limits // bytes
	StackSize        Limits // bytes
	CoreFileSize     Limits // bytes
	ResidentSet      Limits // bytes
	Processes        Limits // processes
	OpenFiles        Limits // files
	LockedMemory     Limits // bytes
	AddressSpace     Limits // bytes
	FileLocks        Limits // locks
	PendingSignals   Limits // signals
	MsgqueueSize     Limits // bytes
	NicePriority     Limits
	RealtimePriority Limits
	RealtimeTimeout  Limits
}

func ReadProcLimits(pid int) (proc ProcLimits, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcLimits(readProcFile(pid, "limits"))
	return
}

func ParseProcLimits(s string) (proc ProcLimits, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcLimits(s)
	return
}

func parseProcLimits(s string) (proc ProcLimits) {
	index := map[string]*Limits{
		"Max cpu time":          &proc.CPUTime,
		"Max file size":         &proc.FileSize,
		"Max data size":         &proc.DataSize,
		"Max stack size":        &proc.StackSize,
		"Max core file size":    &proc.CoreFileSize,
		"Max resident set":      &proc.ResidentSet,
		"Max processes":         &proc.Processes,
		"Max open files":        &proc.OpenFiles,
		"Max locked memory":     &proc.LockedMemory,
		"Max address space":     &proc.AddressSpace,
		"Max file locks":        &proc.FileLocks,
		"Max pending signals":   &proc.PendingSignals,
		"Max msgqueue size":     &proc.MsgqueueSize,
		"Max nice priority":     &proc.NicePriority,
		"Max realtime priority": &proc.RealtimePriority,
		"Max realtime timeout":  &proc.RealtimeTimeout,
	}

	columns := make([]string, 0, 4)
	forEachLineExceptFirst(s, func(line string) {

		columns = columns[:0]
		forEachColumn(line, func(col string) { columns = append(columns, col) })

		var limits Limits
		var length = len(columns)

		if length > 0 {
			limits.Name = columns[0]
		}

		if length > 1 {
			limits.Soft = parseLimitUint(columns[1])
		}

		if length > 2 {
			limits.Hard = parseLimitUint(columns[2])
		}

		if length > 3 {
			limits.Unit = columns[3]
		}

		if ptr := index[limits.Name]; ptr != nil {
			*ptr = limits
		}
	})

	return
}

func parseLimitUint(s string) uint64 {
	if s == "unlimited" {
		return Unlimited
	}
	v, e := strconv.ParseUint(s, 10, 64)
	check(e)
	return v
}
