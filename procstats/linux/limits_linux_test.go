package linux

import (
	"os"
	"testing"
)

func TestReadProcLimits(t *testing.T) {
	if limits, err := ReadProcLimits(os.Getpid()); err != nil {
		t.Error("ReadProcLimits:", err)
	} else if limits.CPUTime.Hard == 0 {
		t.Error("ReadProcLimits: hard cpu time limit cannot be zero")
	}
}
