//go:build linux

package linux

import (
	"os"
	"testing"
)

// These read the live /sys/fs/cgroup files of the machine running the
// benchmark, so only the relative cost between runs is meaningful.

func BenchmarkReadCPUPeriod(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ReadCPUPeriod("")
	}
}

func BenchmarkReadCPUQuota(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ReadCPUQuota("")
	}
}

func BenchmarkReadCPUShares(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ReadCPUShares("")
	}
}

func BenchmarkReadMemoryLimit(b *testing.B) {
	pid := os.Getpid()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ReadMemoryLimit(pid)
	}
}
