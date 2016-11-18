package linux

import (
	"os"
	"testing"
)

func TestGetProcLimits(t *testing.T) {
	if limits, err := GetProcLimits(os.Getpid()); err != nil {
		t.Error("GetProcLimits:", err)
	} else if limits.CPUTime.Hard == 0 {
		t.Error("GetProcLimits: hard cpu time limit cannot be zero")
	}
}
