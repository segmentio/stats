package linux

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func readMemoryLimit(pid int) (limit uint64, err error) {
	if limit = readCGroupMemoryLimit(pid); limit == unlimitedMemoryLimit {
		limit, err = readSysinfoMemoryLimit()
	}
	return limit, err
}

func readCGroupMemoryLimit(pid int) (limit uint64) {
	if cgroups, err := ReadProcCGroup(pid); err == nil {
		limit = readProcCGroupMemoryLimit(cgroups)
	}
	return limit
}

func readProcCGroupMemoryLimit(cgroups ProcCGroup) (limit uint64) {
	if memory, ok := cgroups.Lookup("memory"); ok {
		limit = readMemoryCGroupMemoryLimit(memory)
	}
	return limit
}

func readMemoryCGroupMemoryLimit(cgroup CGroup) (limit uint64) {
	limit = unlimitedMemoryLimit // default value if something doesn't work

	if b, err := os.ReadFile(readMemoryCGroupMemoryLimitFilePath(cgroup.Path)); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64); err == nil {
			limit = v
		}
	}

	return limit
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
	var sysinfo unix.Sysinfo_t

	if err = unix.Sysinfo(&sysinfo); err == nil {
		// unix.Sysinfo returns an uint32 on linux/arm, but uint64 otherwise
		limit = uint64(sysinfo.Unit) * uint64(sysinfo.Totalram)
	}

	return limit, err
}
