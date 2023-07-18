package linux

import (
	"os"
	"testing"
)

func TestReadProcStatm(t *testing.T) {
	if _, err := ReadProcStatm(os.Getpid()); err == nil {
		t.Error("ReadProcStatm should have failed on Darwin")
	}
}
