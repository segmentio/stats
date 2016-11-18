// +build !cgo,!windows

package procstats

func getclktck() uint64 { return 100 }
