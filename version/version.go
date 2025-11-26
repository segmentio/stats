package version

import (
	"runtime"
	"strings"
	"sync"
)

const Version = "5.7.0"

var (
	vsnOnce sync.Once
	vsn     string
)

func isDevel(vsn string) bool {
	return strings.Count(vsn, " ") > 2 || strings.HasPrefix(vsn, "devel")
}

// GoVersion reports the Go version, in a format that is consumable by metrics
// tools.
func GoVersion() string {
	vsnOnce.Do(func() {
		vsn = strings.TrimPrefix(runtime.Version(), "go")
	})
	return vsn
}

// DevelGoVersion reports whether the version of Go that compiled or ran this
// library is a development ("tip") version. This is useful to distinguish
// because tip versions include a commit SHA and change frequently, so are less
// useful for metric reporting.
func DevelGoVersion() bool {
	return isDevel(GoVersion())
}
