package linux

import (
	"os"
	"testing"
)

func TestGetProcSched(t *testing.T) {
	if sched, err := GetProcSched(os.Getpid()); err != nil {
		t.Error("GetProcSched:", err)
	} else if sched.SEAvgLoadSum == 0 {
		t.Error("GetPorcSched: invalid average load sum returned:", sched.SEAvgLoadSum)
	}
}
