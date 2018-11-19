package linux

import (
	"reflect"
	"testing"
)

func TestParseProcCGroup(t *testing.T) {
	text := `11:memory:/user.slice
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

	proc, err := ParseProcCGroup(text)

	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(proc, ProcCGroup{
		{11, "memory", "/user.slice"},
		{10, "pids", "/user.slice/user-1000.slice"},
		{9, "devices", "/user.slice"},
		{8, "net_cls", "/"},
		{8, "net_prio", "/"},
		{7, "blkio", "/user.slice"},
		{6, "hugetlb", "/"},
		{5, "cpu", "/user.slice"},
		{5, "cpuacct", "/user.slice"},
		{4, "perf_event", "/"},
		{3, "freezer", "/"},
		{2, "cpuset", "/"},
		{1, "systemd", "/user.slice/user-1000.slice/session-3925.scope"},
	}) {
		t.Error(proc)
	}
}

func TestProcCGroupLookup(t *testing.T) {
	tests := []struct {
		proc   ProcCGroup
		name   string
		cgroup CGroup
	}{
		{
			proc: ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name: "",
		},
		{
			proc:   ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name:   "A",
			cgroup: CGroup{1, "A", "/"},
		},
		{
			proc:   ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name:   "B",
			cgroup: CGroup{2, "B", "/"},
		},
	}

	for _, test := range tests {
		if cgroup, _ := test.proc.Lookup(test.name); !reflect.DeepEqual(cgroup, test.cgroup) {
			t.Errorf("bad cgroups from name %v: %+v != %+v", test.name, test.cgroup, cgroup)
		}
	}
}

func TestReadCPUPeriod(t *testing.T) {
	period, err := ReadCPUPeriod("")
	if err != nil {
		t.Fatal(err)
	}
	if period == 0 {
		t.Fatal("invalid CPU period:", period)
	}
}

func TestReadCPUQuota(t *testing.T) {
	quota, err := ReadCPUQuota("")
	if err != nil {
		t.Fatal(err)
	}
	if quota == 0 {
		t.Fatal("invalid CPU quota:", quota)
	}
}

func TestReadCPUShares(t *testing.T) {
	shares, err := ReadCPUShares("")
	if err != nil {
		t.Fatal(err)
	}
	if shares == 0 {
		t.Fatal("invalid CPU shares:", shares)
	}
}
