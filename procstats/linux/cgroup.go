package linux

import (
	"strings"
	"time"
)

// ProcCGroup is a type alias for a []CGroup.
type ProcCGroup []CGroup

// CGroup holds configuration information for a Linux cgroup.
type CGroup struct {
	ID   int
	Name string
	Path string // Path in /sys/fs/cgroup
}

// Lookup takes a string argument representing the name of a Linux cgroup
// and returns a CGroup and bool indicating whether or not the cgroup was found.
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

// ReadProcCGroup takes an int argument representing a PID
// and returns a ProcCGroup and error, if any is encountered.
func ReadProcCGroup(pid int) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(readProcFile(pid, "cgroup"))
	return
}

// ParseProcCGroup parses Linux system cgroup data and returns a ProcCGroup and error, if any is encountered.
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

// ReadCPUPeriod takes a string representing a Linux cgroup and returns
// the period as a time.Duration that is applied for this cgroup and an error, if any.
func ReadCPUPeriod(cgroup string) (period time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	period = readCPUPeriod(cgroup)
	return
}

// ReadCPUQuota takes a string representing a Linux cgroup and returns
// the quota as a time.Duration that is applied for this cgroup and an error, if any.
func ReadCPUQuota(cgroup string) (quota time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	quota = readCPUQuota(cgroup)
	return
}

// ReadCPUShares takes a string representing a Linux cgroup and returns
// an int64 representing the cpu shares allotted for this cgroup and an error, if any.
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
