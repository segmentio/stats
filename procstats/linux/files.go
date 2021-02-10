package linux

import "os"

// ReadOpenFileCount takes an int representing a PID and
// returns a uint64 representing the open file descriptor count
// for this process and an error, if any.
func ReadOpenFileCount(pid int) (n uint64, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	n = readOpenFileCount(pid)
	return
}

func readOpenFileCount(pid int) uint64 {
	f, err := os.Open(procPath(pid, "fd"))
	check(err)
	defer f.Close()

	// May not be the most efficient way to do this, but the little dance
	// for getting open/readdirent/parsedirent is gonna take a while to get
	// right.
	s, err := f.Readdirnames(-1)
	check(err)
	return uint64(len(s))
}
