package linux

import "testing"

const (
	procCGroupV1 = `11:memory:/user.slice
10:pids:/user.slice/user-1000.slice
9:devices:/user.slice
8:net_cls,net_prio:/
7:blkio:/user.slice
6:hugetlb:/
5:cpu,cpuacct:/user.slice
4:perf_event:/
3:freezer:/
2:cpuset:/
1:name=systemd:/user.slice/user-1000.slice/session-3925.scope
`
	procCGroupV2 = "0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod1.slice/cri-containerd-abc.scope\n"
)

func BenchmarkParseProcCGroup(b *testing.B) {
	for name, text := range map[string]string{"v1": procCGroupV1, "v2": procCGroupV2} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := ParseProcCGroup(text); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
