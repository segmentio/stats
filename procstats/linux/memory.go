package linux

const (
	unlimitedMemoryLimit = 9223372036854771712
)

func ReadMemoryLimit(pid int) (limit uint64, err error) { return readMemoryLimit(pid) }
