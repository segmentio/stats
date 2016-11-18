package linux

func getMemoryLimit(pid int) (limit uint64, err error) {
	limit = unlimitedMemoryLimit
	return
}
