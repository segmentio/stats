package linux

import (
	"os"
	"testing"
)

func TestGetProcStatm(t *testing.T) {
	if statm, err := GetProcStatm(os.Getpid()); err != nil {
		t.Error("GetProcStatm:", err)
	} else if statm.Size == 0 {
		t.Error("GetPorcStatm: size cannot be zero")
	}
}
