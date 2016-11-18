package linux

import (
	"reflect"
	"testing"
)

func TestParseProcLimits(t *testing.T) {
	text := `Limit                     Soft Limit           Hard Limit           Units
Max cpu time              unlimited            unlimited            seconds
Max file size             unlimited            unlimited            bytes
Max data size             unlimited            unlimited            bytes
Max stack size            8388608              unlimited            bytes
Max core file size        0                    unlimited            bytes
Max resident set          unlimited            unlimited            bytes
Max processes             unlimited            unlimited            processes
Max open files            1048576              1048576              files
Max locked memory         65536                65536                bytes
Max address space         unlimited            unlimited            bytes
Max file locks            unlimited            unlimited            locks
Max pending signals       7767                 7767                 signals
Max msgqueue size         819200               819200               bytes
Max nice priority         0                    0
Max realtime priority     0                    0
Max realtime timeout      unlimited            unlimited            us
`

	proc, err := ParseProcLimits(text)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(proc, ProcLimits{
		CPUTime:          Limits{"Max cpu time", Unlimited, Unlimited, "seconds"},
		FileSize:         Limits{"Max file size", Unlimited, Unlimited, "bytes"},
		DataSize:         Limits{"Max data size", Unlimited, Unlimited, "bytes"},
		StackSize:        Limits{"Max stack size", 8388608, Unlimited, "bytes"},
		CoreFileSize:     Limits{"Max core file size", 0, Unlimited, "bytes"},
		ResidentSet:      Limits{"Max resident set", Unlimited, Unlimited, "bytes"},
		Processes:        Limits{"Max processes", Unlimited, Unlimited, "processes"},
		OpenFiles:        Limits{"Max open files", 1048576, 1048576, "files"},
		LockedMemory:     Limits{"Max locked memory", 65536, 65536, "bytes"},
		AddressSpace:     Limits{"Max address space", Unlimited, Unlimited, "bytes"},
		FileLocks:        Limits{"Max file locks", Unlimited, Unlimited, "locks"},
		PendingSignals:   Limits{"Max pending signals", 7767, 7767, "signals"},
		MsgqueueSize:     Limits{"Max msgqueue size", 819200, 819200, "bytes"},
		NicePriority:     Limits{"Max nice priority", 0, 0, ""},
		RealtimePriority: Limits{"Max realtime priority", 0, 0, ""},
		RealtimeTimeout:  Limits{"Max realtime timeout", Unlimited, Unlimited, "us"},
	}) {
		t.Error(proc)
	}
}
