package linux

import (
	"os"
	"testing"
)

func TestReadMemoryLimit(t *testing.T) {
	if limit, err := ReadMemoryLimit(os.Getpid()); err != nil {
		t.Error(err)

	} else if limit == 0 {
		t.Error("memory should not be zero")

	} else if limit == unlimitedMemoryLimit {
		t.Error("memory should not be unlimited")
	}
}
