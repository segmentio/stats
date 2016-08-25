package linux

import "strings"

type ProcCGroup []CGroup

type CGroup struct {
	ID   int
	Name string
	Path string // Path in /sys/fs/cgroup
}

func (pcg ProcCGroup) GetByID(id int) []CGroup {
	return pcg.GetByFunc(func(cg CGroup) bool { return cg.ID == id })
}

func (pcg ProcCGroup) GetByName(name string) []CGroup {
	return pcg.GetByFunc(func(cg CGroup) bool { return cg.Name == name })
}

func (pcg ProcCGroup) GetByFunc(f func(CGroup) bool) (res []CGroup) {
	for _, cg := range pcg {
		if f(cg) {
			res = append(res, cg)
		}
	}
	return
}

func GetProcCGroup(pid int) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(readProcFile(pid, "cgroup"))
	return
}

func ParseProcCGroup(s string) (proc ProcCGroup, err error) {
	defer func() { err = convertPanicToError(recover()) }()
	proc = parseProcCGroup(s)
	return
}

func parseProcCGroup(s string) (proc ProcCGroup) {
	forEachLine(s, func(line string) {
		id, line := split(line, ':')
		name, path := split(line, ':')

		for len(name) != 0 {
			var next string
			name, next = split(name, ',')

			if strings.HasPrefix(name, "name=") { // WTF?
				name = strings.TrimSpace(name[5:])
			}

			proc = append(proc, CGroup{ID: atoi(id), Name: name, Path: path})
			name = next
		}
	})
	return
}
