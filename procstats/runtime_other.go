// +build !linux,!darwin

package procstats

func threadCount() uint64 { return 0 }

func fdCount() uint64 { return 0 }

func fdMax() uint64 { return 0 }
