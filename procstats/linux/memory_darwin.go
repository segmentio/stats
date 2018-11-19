package linux

func readMemoryLimit(pid int) (limit uint64, err error) {
	limit = unlimitedMemoryLimit
	return
}
