package linux

import (
	"os"
	"testing"
)

func TestGetProcStat(t *testing.T) {
	if _, err := GetProcStat(os.Getpid()); err == nil {
		t.Error("GetProcStat should have failed on Darwin")
	}
}
