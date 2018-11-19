package linux

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func readMemoryLimit(pid int) (limit uint64, err error) {
	if limit = readCGroupMemoryLimit(pid); limit == unlimitedMemoryLimit {
		limit, err = readSysinfoMemoryLimit()
	}
	return
}

func readCGroupMemoryLimit(pid int) (limit uint64) {
	if cgroups, err := ReadProcCGroup(pid); err == nil {
		limit = readProcCGroupMemoryLimit(cgroups)
	}
	return
}

func readProcCGroupMemoryLimit(cgroups ProcCGroup) (limit uint64) {
	if memory, ok := cgroups.Lookup("memory"); ok {
		limit = readMemoryCGroupMemoryLimit(memory)
	}
	return
}

func readMemoryCGroupMemoryLimit(cgroup CGroup) (limit uint64) {
	limit = unlimitedMemoryLimit // default value if something doesn't work

	if b, err := ioutil.ReadFile(readMemoryCGroupMemoryLimitFilePath(cgroup.Path)); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64); err == nil {
			limit = v
		}
	}

	return
}

func readMemoryCGroupMemoryLimitFilePath(cgroupPath string) string {
	path := "/sys/fs/cgroup/memory"

	// Docker generates weird cgroup paths that don't really exist on the file
	// system.
	if !strings.HasPrefix(cgroupPath, "/docker/") {
		path = filepath.Join(path, cgroupPath)
	}

	return filepath.Join(path, "memory.limit_in_bytes")
}

func readSysinfoMemoryLimit() (limit uint64, err error) {
	var sysinfo syscall.Sysinfo_t

	if err = syscall.Sysinfo(&sysinfo); err == nil {
		limit = uint64(sysinfo.Unit) * sysinfo.Totalram
	}

	return
}
