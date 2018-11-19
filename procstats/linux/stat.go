package linux

import "fmt"

type ProcState rune

const (
	Running                         ProcState = 'R'
	Sleeping                        ProcState = 'S'
	WaitingUninterruptibleDiskSleep ProcState = 'D'
	Zombie                          ProcState = 'Z'
	Stopped                         ProcState = 'T'
	TracingStop                     ProcState = 't'
	Paging                          ProcState = 'P'
	Dead                            ProcState = 'X'
	Dead_                           ProcState = 'x'
	Wakekill                        ProcState = 'W'
	Parked                          ProcState = 'P'
)

func (ps *ProcState) Scan(s fmt.ScanState, _ rune) (err error) {
	var c rune
	s.SkipSpace()

	if c, _, err = s.ReadRune(); err == nil {
		*ps = ProcState(c)
	}

	return
}

type ProcStat struct {
	Pid                 int32     // (1) pid
	Comm                string    // (2) comm
	State               ProcState // (3) state
	Ppid                int32     // (4) ppid
	Pgrp                int32     // (5) prgp
	Session             int32     // (6) session
	TTY                 int32     // (7) tty_nr
	Tpgid               int32     // (8) tpgid
	Flags               uint32    // (9) flags
	Minflt              uint64    // (10) minflt
	Cminflt             uint64    // (11) cminflt
	Majflt              uint64    // (12) majflt
	Cmajflt             uint64    // (13) cmajflt
	Utime               uint64    // (14) utime
	Stime               uint64    // (15) stime
	Cutime              int64     // (16) cutime
	Cstime              int64     // (17) cstime
	Priority            int64     // (18) priority
	Nice                int64     // (19) nice
	NumThreads          int64     // (20) num_threads
	Itrealvalue         int64     // (21) itrealvalue
	Starttime           uint64    // (22) starttime
	Vsize               uint64    // (23) vsize
	Rss                 uint64    // (24) rss
	Rsslim              uint64    // (25) rsslim
	Startcode           uintptr   // (26) startcode
	Endcode             uintptr   // (27) endcode
	Startstack          uintptr   // (28) startstack
	Kstkeep             uint64    // (29) kstkeep
	Kstkeip             uint64    // (30) kstkeip
	Signal              uint64    // (31) signal
	Blocked             uint64    // (32) blocked
	Sigignore           uint64    // (33) sigignore
	Sigcatch            uint64    // (34) sigcatch
	Wchan               uintptr   // (35) wchan
	Nswap               uint64    // (36) nswap
	Cnswap              uint64    // (37) cnswap
	ExitSignal          int32     // (38) exit_signal
	Processor           int32     // (39) processor
	RTPriority          uint32    // (40) rt_priority
	Policy              uint32    // (41) policy
	DelayacctBlkioTicks uint64    // (42) delayacct_blkio_ticks
	GuestTime           uint64    // (43) guest_time
	CguestTime          int64     // (44) cguest_time
	StartData           uintptr   // (45) start_data
	EndData             uintptr   // (46) end_data
	StartBrk            uintptr   // (47) start_brk
	ArgStart            uintptr   // (48) arg_start
	ArgEnd              uintptr   // (49) arg_end
	EnvStart            uintptr   // (50) env_start
	EnvEnd              uintptr   // (51) env_end
	ExitCode            int32     // (52) exit_code
}

func ReadProcStat(pid int) (proc ProcStat, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	return ParseProcStat(readProcFile(pid, "stat"))
}

func ParseProcStat(s string) (proc ProcStat, err error) {
	_, err = fmt.Sscan(s,
		&proc.Pid,
		&proc.Comm,
		&proc.State,
		&proc.Ppid,
		&proc.Pgrp,
		&proc.Session,
		&proc.TTY,
		&proc.Tpgid,
		&proc.Flags,
		&proc.Minflt,
		&proc.Cminflt,
		&proc.Majflt,
		&proc.Cmajflt,
		&proc.Utime,
		&proc.Stime,
		&proc.Cutime,
		&proc.Cstime,
		&proc.Priority,
		&proc.Nice,
		&proc.NumThreads,
		&proc.Itrealvalue,
		&proc.Starttime,
		&proc.Vsize,
		&proc.Rss,
		&proc.Rsslim,
		&proc.Startcode,
		&proc.Endcode,
		&proc.Startstack,
		&proc.Kstkeep,
		&proc.Kstkeip,
		&proc.Signal,
		&proc.Blocked,
		&proc.Sigignore,
		&proc.Sigcatch,
		&proc.Wchan,
		&proc.Nswap,
		&proc.Cnswap,
		&proc.ExitSignal,
		&proc.Processor,
		&proc.RTPriority,
		&proc.Policy,
		&proc.DelayacctBlkioTicks,
		&proc.GuestTime,
		&proc.CguestTime,
		&proc.StartData,
		&proc.EndData,
		&proc.StartBrk,
		&proc.ArgStart,
		&proc.ArgEnd,
		&proc.EnvStart,
		&proc.EnvEnd,
		&proc.ExitCode,
	)
	return
}
