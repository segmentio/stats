package linux

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func getMemoryLimit(pid int) (limit uint64, err error) {
	if limit = getCGroupMemoryLimit(pid); limit == unlimitedMemoryLimit {
		limit, err = getSysinfoMemoryLimit()
	}
	return
}

func getCGroupMemoryLimit(pid int) (limit uint64) {
	if cgroups, err := GetProcCGroup(pid); err == nil {
		limit = getProcCGroupMemoryLimit(cgroups)
	}
	return
}

func getProcCGroupMemoryLimit(cgroups ProcCGroup) (limit uint64) {
	if memory := cgroups.GetByName("memory"); len(memory) != 0 {
		limit = getMemoryCGroupMemoryLimit(memory[0]) // can there be more?
	}
	return
}

func getMemoryCGroupMemoryLimit(cgroup CGroup) (limit uint64) {
	limit = unlimitedMemoryLimit // default value if something doesn't work

	if b, err := ioutil.ReadFile(getMemoryCGroupMemoryLimitFilePath(cgroup.Path)); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64); err == nil {
			limit = v
		}
	}

	return
}

func getMemoryCGroupMemoryLimitFilePath(cgroupPath string) string {
	path := "/sys/fs/cgroup/memory"

	// Docker generates weird cgroup paths that don't really exist on the file
	// system.
	if !strings.HasPrefix(cgroupPath, "/docker/") {
		path = filepath.Join(path, cgroupPath)
	}

	return filepath.Join(path, "memory.limit_in_bytes")
}

func getSysinfoMemoryLimit() (limit uint64, err error) {
	var sysinfo syscall.Sysinfo_t

	if err = syscall.Sysinfo(&sysinfo); err == nil {
		limit = uint64(sysinfo.Unit) * sysinfo.Totalram
	}

	return
}
