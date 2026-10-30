//go:build !linux

package linux

func readSmallFile(path string) string {
	return readFile(path)
}
