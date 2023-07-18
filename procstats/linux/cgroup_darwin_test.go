package linux

import (
	"os"
	"testing"
)

func TestReadProcCGroup(t *testing.T) {
	if _, err := ReadProcCGroup(os.Getpid()); err == nil {
		t.Error("ReadProcCGroup should have failed on Darwin")
	}
}
