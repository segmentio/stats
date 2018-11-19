package linux

import (
	"os"
	"testing"
)

func TestReadProcStat(t *testing.T) {
	if stat, err := ReadProcStat(os.Getpid()); err != nil {
		t.Error("ReadProcStat:", err)
	} else if int(stat.Pid) != os.Getpid() {
		t.Error("ReadPorcStat: invalid pid returned:", stat.Pid)
	}
}
