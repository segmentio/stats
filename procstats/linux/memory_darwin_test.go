package linux

import (
	"os"
	"testing"
)

func TestGetMemoryLimit(t *testing.T) {
	if limit, err := GetMemoryLimit(os.Getpid()); err != nil || limit != unlimitedMemoryLimit {
		t.Error("memory should be unlimited on darwin")
	}
}
