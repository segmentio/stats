package linux

import (
	"os"
	"testing"
)

func TestGetProcStatm(t *testing.T) {
	if _, err := GetProcStatm(os.Getpid()); err == nil {
		t.Error("GetProcStatm should have failed on Darwin")
	}
}
