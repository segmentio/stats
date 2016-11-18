package linux

const (
	unlimitedMemoryLimit = 9223372036854771712
)

func GetMemoryLimit(pid int) (limit uint64, err error) { return getMemoryLimit(pid) }
