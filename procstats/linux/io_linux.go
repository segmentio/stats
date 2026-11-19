package linux

import "golang.org/x/sys/unix"

// readSmallFile reads a file that fits in a single small read, such as a
// cgroup or proc setting, with fewer syscalls and allocations than
// os.ReadFile. It falls back to readFile on any error or if the file may be
// larger than the buffer.
func readSmallFile(path string) string {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return readFile(path) // reports the error the same way as before
	}
	var buf [512]byte
	n, err := unix.Read(fd, buf[:])
	unix.Close(fd)
	if err != nil || n == len(buf) {
		return readFile(path)
	}
	return string(buf[:n])
}
