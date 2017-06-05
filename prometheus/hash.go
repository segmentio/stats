package prometheus

// much better throughput because it doesn't force memory allocation to mask
// the hash sum into an interface and convert the string to a byte slice.
const (
	// FNV-1a
	offset64 = uint64(14695981039346656037)
	prime64  = uint64(1099511628211)
)

func hash(s string) uint64 {
	return hashS(offset64, s)
}

func hashS(h uint64, s string) uint64 {
	for i := range s {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}
