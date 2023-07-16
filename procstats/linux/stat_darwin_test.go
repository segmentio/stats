package linux

import (
	"os"
	"testing"
)

func TestReadProcStat(t *testing.T) {
	if _, err := ReadProcStat(os.Getpid()); err == nil {
		t.Error("ReadProcStat should have failed on Darwin")
	}
}
