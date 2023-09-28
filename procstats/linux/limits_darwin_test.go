package linux

import (
	"os"
	"testing"
)

func TestReadProcLimits(t *testing.T) {
	if _, err := ReadProcLimits(os.Getpid()); err == nil {
		t.Error("ReadProcLimits should have failed on Darwin")
	}
}
