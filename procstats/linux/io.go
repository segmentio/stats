package linux

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func readFile(path string) string {
	b, err := os.ReadFile(path)
	check(err)
	return string(b)
}

func readProcFile(who any, what string) string {
	return readFile(procPath(who, what))
}

func readMicrosecondFile(path string) time.Duration {
	return time.Duration(readIntFile(path)) * time.Microsecond
}

func readIntFile(path string) int64 {
	return parseInt(strings.TrimSpace(readSmallFile(path)))
}

func parseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	check(err)
	return i
}

func procPath(who any, what string) string {
	return filepath.Join("/proc", fmt.Sprint(who), what)
}

func cgroupPath(dir, cgroup, file string) string {
	return filepath.Join("/sys/fs/cgroup", dir, cgroup, file)
}

// isCGroupV2 reports whether /sys/fs/cgroup is the cgroup v2 unified hierarchy.
// The hierarchy cannot change while the process runs, so it is checked once.
var isCGroupV2 = sync.OnceValue(func() bool {
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil
})
