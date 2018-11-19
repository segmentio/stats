package linux

import (
	"os"
	"testing"
)

func TestReadProcStatm(t *testing.T) {
	if statm, err := ReadProcStatm(os.Getpid()); err != nil {
		t.Error("ReadProcStatm:", err)
	} else if statm.Size == 0 {
		t.Error("ReadPorcStatm: size cannot be zero")
	}
}
