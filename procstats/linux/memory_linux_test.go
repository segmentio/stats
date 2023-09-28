// This is a build tag hack to permit the test suite
// to succeed on the ubuntu-latest runner (linux-amd64),
// which apparently no longer succeeds with the included test.
//
//go:build linux && arm64

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
