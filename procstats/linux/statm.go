package linux

import "fmt"

// ProcStatm contains statistics about memory utilization of a process.
type ProcStatm struct {
	Size     uint64 // (1) size
	Resident uint64 // (2) resident
	Share    uint64 // (3) share
	Text     uint64 // (4) text
	Lib      uint64 // (5) lib
	Data     uint64 // (6) data
	Dt       uint64 // (7) dt
}

// ReadProcStatm returns a ProcStatm and an error, if any, for a PID.
func ReadProcStatm(pid int) (proc ProcStatm, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	return ParseProcStatm(readProcFile(pid, "statm"))
}

// ParseProcStatm parses system proc data and returns a ProcStatm and error, if any.
func ParseProcStatm(s string) (proc ProcStatm, err error) {
	_, err = fmt.Sscan(s,
		&proc.Size,
		&proc.Resident,
		&proc.Share,
		&proc.Text,
		&proc.Lib,
		&proc.Data,
		&proc.Dt,
	)
	return
}
