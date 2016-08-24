package linux

import (
	"os"
	"testing"
)

func TestGetProcLimits(t *testing.T) {
	if _, err := GetProcLimits(os.Getpid()); err == nil {
		t.Error("GetProcLimits should have failed on Darwin")
	}
}
