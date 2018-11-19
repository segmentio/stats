package linux

import "os"

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
