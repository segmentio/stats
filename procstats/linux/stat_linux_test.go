package linux

import (
	"os"
	"testing"
)

func TestGetProcStat(t *testing.T) {
	if stat, err := GetProcStat(os.Getpid()); err != nil {
		t.Error("GetProcStat:", err)
	} else if int(stat.Pid) != os.Getpid() {
		t.Error("GetPorcStat: invalid pid returned:", stat.Pid)
	}
}
