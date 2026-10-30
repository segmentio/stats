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
	return cgroup, ok
}

// Unified returns the cgroup v2 (unified hierarchy) entry, which
// /proc/<pid>/cgroup reports as "0::<path>", and a bool indicating whether or
// not it was found.
func (pcg ProcCGroup) Unified() (cgroup CGroup, ok bool) {
	for _, cg := range pcg {
		if cg.ID == 0 && cg.Name == "" {
			return cg, true
		}
	}
	return CGroup{}, false
}

// ReadProcCGroup takes an int argument representing a PID
// and returns a ProcCGroup and error, if any is encountered.
func ReadProcCGroup(pid int) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(readProcFile(pid, "cgroup"))
	return proc, err
}

// ParseProcCGroup parses Linux system cgroup data and returns a ProcCGroup and error, if any is encountered.
func ParseProcCGroup(s string) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(s)
	return proc, err
}

func parseProcCGroup(s string) (proc ProcCGroup) {
	forEachLine(s, func(line string) {
		id, line := split(line, ':')
		name, path := split(line, ':')

		if id == "0" && name == "" { // cgroup v2 unified hierarchy
			proc = append(proc, CGroup{ID: 0, Name: "", Path: path})
			return
		}

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
	return proc
}

// ReadCPUPeriod takes a string representing a Linux cgroup and returns
// the period as a time.Duration that is applied for this cgroup and an error, if any.
func ReadCPUPeriod(cgroup string) (period time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	period = readCPUPeriod(cgroup)
	return period, err
}

// ReadCPUQuota takes a string representing a Linux cgroup and returns
// the quota as a time.Duration that is applied for this cgroup and an error, if any.
func ReadCPUQuota(cgroup string) (quota time.Duration, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	quota = readCPUQuota(cgroup)
	return quota, err
}

// ReadCPUShares takes a string representing a Linux cgroup and returns
// an int64 representing the cpu shares allotted for this cgroup and an error, if any.
func ReadCPUShares(cgroup string) (shares int64, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	shares = readCPUShares(cgroup)
	return shares, err
}

func readCPUPeriod(cgroup string) time.Duration {
	if isCGroupV2() {
		_, period := readCPUMax(cgroup)
		return period
	}
	return readMicrosecondFile(cgroupPath("cpu", cgroup, "cpu.cfs_period_us"))
}

func readCPUQuota(cgroup string) time.Duration {
	if isCGroupV2() {
		quota, _ := readCPUMax(cgroup)
		return quota
	}
	return readMicrosecondFile(cgroupPath("cpu", cgroup, "cpu.cfs_quota_us"))
}

func readCPUShares(cgroup string) int64 {
	if isCGroupV2() {
		return weightToShares(readIntFile(cgroupPath("", cgroup, "cpu.weight")))
	}
	return readIntFile(cgroupPath("cpu", cgroup, "cpu.shares"))
}

func readCPUMax(cgroup string) (quota, period time.Duration) {
	return parseCPUMax(readFile(cgroupPath("", cgroup, "cpu.max")))
}

// parseCPUMax parses the cgroup v2 cpu.max format, "<quota> <period>", where
// quota is "max" when unlimited. An unlimited quota is reported as -1µs to
// match what cgroup v1 reports in cpu.cfs_quota_us.
func parseCPUMax(s string) (quota, period time.Duration) {
	q, p := split(strings.TrimSpace(s), ' ')
	if q == "max" {
		quota = -1 * time.Microsecond
	} else {
		quota = time.Duration(parseInt(q)) * time.Microsecond
	}
	period = time.Duration(parseInt(p)) * time.Microsecond
	return quota, period
}

// weightToShares converts a cgroup v2 cpu.weight (1-10000) to the equivalent
// cgroup v1 cpu.shares (2-262144), using the inverse of the conversion runc and
// the kubelet apply when writing cpu.weight.
func weightToShares(weight int64) int64 {
	if weight <= 0 {
		return 0
	}
	return 2 + (weight-1)*262142/9999
}
