package linux

import "fmt"

type ProcStatm struct {
	Size     uint64 // (1) size
	Resident uint64 // (2) resident
	Share    uint64 // (3) share
	Text     uint64 // (4) text
	Lib      uint64 // (5) lib
	Data     uint64 // (6) data
	Dt       uint64 // (7) dt
}

func ReadProcStatm(pid int) (proc ProcStatm, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	return ParseProcStatm(readProcFile(pid, "statm"))
}

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
