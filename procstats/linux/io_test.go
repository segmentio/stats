package linux

import (
	"io/ioutil"
	"testing"
)

func TestReadFile(t *testing.T) {
	f, _ := ioutil.TempFile("/tmp", "io_test_")
	path := f.Name()
	f.Write([]byte("Hello World!\n"))
	f.Close()

	if s := readFile(path); s != "Hello World!\n" {
		t.Error("invalid file content:", s)
	}
}

func TestReadProcFilePanic(t *testing.T) {
	defer func() { recover() }()
	readProcFile("asdfasdf", "asdfasdf") // doesn't exist
	t.Error("should have raised a panic")
}
