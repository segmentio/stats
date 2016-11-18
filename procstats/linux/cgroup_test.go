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

func TestProcCGroupGetByID(t *testing.T) {
	tests := []struct {
		proc    ProcCGroup
		id      int
		cgroups []CGroup
	}{
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			id:      0,
			cgroups: nil,
		},
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			id:      1,
			cgroups: []CGroup{{1, "A", "/"}},
		},
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			id:      2,
			cgroups: []CGroup{{2, "B", "/"}, {2, "C", "/"}},
		},
	}

	for _, test := range tests {
		if cgroups := test.proc.GetByID(test.id); !reflect.DeepEqual(cgroups, test.cgroups) {
			t.Errorf("bad cgroups from id %v: %v != %v", test.id, test.cgroups, cgroups)
		}
	}
}

func TestProcCGroupGetByName(t *testing.T) {
	tests := []struct {
		proc    ProcCGroup
		name    string
		cgroups []CGroup
	}{
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name:    "",
			cgroups: nil,
		},
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name:    "A",
			cgroups: []CGroup{{1, "A", "/"}},
		},
		{
			proc:    ProcCGroup{{1, "A", "/"}, {2, "B", "/"}, {2, "C", "/"}},
			name:    "B",
			cgroups: []CGroup{{2, "B", "/"}},
		},
	}

	for _, test := range tests {
		if cgroups := test.proc.GetByName(test.name); !reflect.DeepEqual(cgroups, test.cgroups) {
			t.Errorf("bad cgroups from name %v: %v != %v", test.name, test.cgroups, cgroups)
		}
	}
}
