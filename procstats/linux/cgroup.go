package linux

import (
	"strings"
	"time"
)

type ProcCGroup []CGroup

type CGroup struct {
	ID   int
	Name string
	Path string // Path in /sys/fs/cgroup
}

func (pcg ProcCGroup) Lookup(name string) (cgroup CGroup, ok bool) {
	forEachToken(name, ",", func(key1 string) {
		for _, cg := range pcg {
			forEachToken(cg.Name, ",", func(key2 string) {
				if key1 == key2 {
					cgroup, ok = cg, true
				}
			})
		}
	})
	return
}

func ReadProcCGroup(pid int) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(readProcFile(pid, "cgroup"))
	return
}

func ParseProcCGroup(s string) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(s)
	return
}

func parseProcCGroup(s string) (proc ProcCGroup) {
	forEachLine(s, func(line string) {
		id, line := split(line, ':')
		name, path := split(line, ':')

		for len(name) != 0 {
			var next string
			name, next = split(name, ',')

			if strings.HasPrefix(name, "name=") { // WTF?
				name = strings.TrimSpace(name[5:])
			}

			proc = append(proc, CGroup{ID: atoi(id), Name: name, Path: path})
			name = next
		}
	})
	return
}

func ReadCPUPeriod(cgroup string) (period time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	period = readCPUPeriod(cgroup)
	return
}

func ReadCPUQuota(cgroup string) (quota time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	quota = readCPUQuota(cgroup)
	return
}

func ReadCPUShares(cgroup string) (shares int64, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	shares = readCPUShares(cgroup)
	return
}

func readCPUPeriod(cgroup string) time.Duration {
	return readMicrosecondFile(cgroupPath("cpu", cgroup, "cpu.cfs_period_us"))
}

func readCPUQuota(cgroup string) time.Duration {
	return readMicrosecondFile(cgroupPath("cpu", cgroup, "cpu.cfs_quota_us"))
}

func readCPUShares(cgroup string) int64 {
	return readIntFile(cgroupPath("cpu", cgroup, "cpu.shares"))
}
