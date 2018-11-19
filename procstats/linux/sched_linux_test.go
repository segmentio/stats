package linux

import (
	"os"
	"testing"
)

func TestReadProcSched(t *testing.T) {
	if _, err := ReadProcSched(os.Getpid()); err != nil {
		t.Error("ReadProcSched:", err)
	}
}
