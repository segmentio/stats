package linux

import (
	"os"
	"testing"
)

func TestReadMemoryLimit(t *testing.T) {
	limit, err := ReadMemoryLimit(os.Getpid())
	if err != nil || limit != unlimitedMemoryLimit {
		t.Error("memory should be unlimited on darwin")
	}
}
