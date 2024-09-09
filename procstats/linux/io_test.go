package linux

import (
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	f, err := os.CreateTemp("/tmp", "io_test_")
	if err != nil {
		t.Error(err)
	}
	path := f.Name()
	if _, err := f.Write([]byte("Hello World!\n")); err != nil {
		t.Error(err)
	}
	if err := f.Close(); err != nil {
		t.Error(err)
	}
	if s := readFile(path); s != "Hello World!\n" {
		t.Error("invalid file content:", s)
	}
}

func TestReadProcFilePanic(t *testing.T) {
	defer func() { recover() }()
	readProcFile("asdfasdf", "asdfasdf") // doesn't exist
	t.Error("should have raised a panic")
}
