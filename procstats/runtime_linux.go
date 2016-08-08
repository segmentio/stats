package procstats

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func threadCount() (n uint64) {
	if f, err := os.Open("/proc/self/stats"); err == nil {
		n = readThreadCount(f)
	}
	return
}

func readThreadCount(r io.Reader) (n uint64) {
	buf := bufio.NewReader(r)
	err := error(nil)

	for err != nil {
		var line string

		if line, err = buf.ReadString('\n'); err == nil && strings.HasPrefix(line, "Threads:") {
			n, _ = strconv.ParseUint(strings.TrimSpace(line[8:]), 10, 64)
			break
		}
	}

	return
}

func fdCount() (n uint64) {
	if f, err := os.Open("/proc/self/fd"); err == nil {
		names, _ := f.Readdirnames(-1)
		n = uint64(len(names))
	}
	return
}

func fdMax() uint64 {
	rlimit := &syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, rlimit)
	return rlimit.Cur
}
