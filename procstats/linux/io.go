package linux

import (
	"fmt"
	"io/ioutil"
)

func readFile(path string) string {
	b, e := ioutil.ReadFile(path)
	check(e)
	return string(b)
}

func readProcFile(who interface{}, what string) string {
	return readFile(procPath(who, what))
}

func procPath(who interface{}, what string) string {
	return fmt.Sprintf("/proc/%v/%s", who, what)
}
