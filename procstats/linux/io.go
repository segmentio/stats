package linux

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func readFile(path string) string {
	b, err := ioutil.ReadFile(path)
	check(err)
	return string(b)
}

func readProcFile(who interface{}, what string) string {
	return readFile(procPath(who, what))
}

func readMicrosecondFile(path string) time.Duration {
	return time.Duration(readIntFile(path)) * time.Microsecond
}

func readIntFile(path string) int64 {
	return parseInt(strings.TrimSpace(readFile(path)))
}

func parseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	check(err)
	return i
}

func procPath(who interface{}, what string) string {
	return filepath.Join("/proc", fmt.Sprint(who), what)
}

func cgroupPath(dir, cgroup, file string) string {
	return filepath.Join("/sys/fs/cgroup", dir, cgroup, file)
}
