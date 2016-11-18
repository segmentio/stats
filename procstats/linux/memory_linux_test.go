package linux

import (
	"os"
	"testing"
)

func TestGetMemoryLimit(t *testing.T) {
	if limit, err := GetMemoryLimit(os.Getpid()); err != nil {
		t.Error(err)

	} else if limit == 0 {
		t.Error("memory should not be zero")

	} else if limit == unlimitedMemoryLimit {
		t.Error("memory should not be unlimited")
	}
}
