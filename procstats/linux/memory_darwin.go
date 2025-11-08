package linux

func readMemoryLimit(_ int) (limit uint64, err error) {
	limit = unlimitedMemoryLimit
	return limit, err
}
