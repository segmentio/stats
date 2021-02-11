package linux

const (
	unlimitedMemoryLimit = 9223372036854771712
)

// ReadMemoryLimit returns the memory limit and an error, if any, for a PID.
func ReadMemoryLimit(pid int) (limit uint64, err error) { return readMemoryLimit(pid) }
