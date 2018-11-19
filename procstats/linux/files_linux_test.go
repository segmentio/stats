package linux

import (
	"os"
	"testing"
)

func TestReadOpenFileCount(t *testing.T) {
	if count, err := ReadOpenFileCount(os.Getpid()); err != nil {
		t.Error("ReadOpenFileCount:", err)
	} else if count == 0 {
		t.Error("ReadOpenFileCount: cannot return zero")
	}
}
