package linux

import (
	"reflect"
	"testing"
)

func TestParseProcStat(t *testing.T) {
	text := `69 (cat) R 56 1 1 0 -1 4210944 83 0 0 0 0 0 0 0 20 0 1 0 1977676 4644864 193 18446744073709551615 4194304 4240332 140724300789216 140724300788568 140342654634416 0 0 0 0 0 0 0 17 0 0 0 0 0 0 6340112 6341364 24690688 140724300791495 140724300791515 140724300791515 140724300791791 0`

	proc, err := ParseProcStat(text)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(proc, ProcStat{
		Pid:                 69,
		Comm:                "(cat)",
		State:               Running,
		Ppid:                56,
		Pgrp:                1,
		Session:             1,
		TTY:                 0,
		Tpgid:               -1,
		Flags:               4210944,
		Minflt:              83,
		Cminflt:             0,
		Majflt:              0,
		Cmajflt:             0,
		Utime:               0,
		Stime:               0,
		Cutime:              0,
		Cstime:              0,
		Priority:            20,
		Nice:                0,
		NumThreads:          1,
		Itrealvalue:         0,
		Starttime:           1977676,
		Vsize:               4644864,
		Rss:                 193,
		Rsslim:              18446744073709551615,
		Startcode:           4194304,
		Endcode:             4240332,
		Startstack:          140724300789216,
		Kstkeep:             140724300788568,
		Kstkeip:             140342654634416,
		Signal:              0,
		Blocked:             0,
		Sigignore:           0,
		Sigcatch:            0,
		Wchan:               0,
		Nswap:               0,
		Cnswap:              0,
		ExitSignal:          17,
		Processor:           0,
		RTPriority:          0,
		Policy:              0,
		DelayacctBlkioTicks: 0,
		GuestTime:           0,
		CguestTime:          0,
		StartData:           6340112,
		EndData:             6341364,
		StartBrk:            24690688,
		ArgStart:            140724300791495,
		ArgEnd:              140724300791515,
		EnvStart:            140724300791515,
		EnvEnd:              140724300791791,
		ExitCode:            0,
	}) {
		t.Error(proc)
	}
}
