package linux

import (
	"os"
	"testing"
)

func TestReadOpenFileCount(t *testing.T) {
	if _, err := ReadOpenFileCount(os.Getpid()); err == nil {
		t.Error("ReadOpenFileCount should have failed on Darwin")
	}
}
