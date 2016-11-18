package linux

import (
	"os"
	"testing"
)

func TestGetOpenFileCount(t *testing.T) {
	if count, err := GetOpenFileCount(os.Getpid()); err != nil {
		t.Error("GetOpenFileCount:", err)
	} else if count == 0 {
		t.Error("GetOpenFileCount: cannot return zero")
	}
}
