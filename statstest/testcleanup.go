package statstest

import (
	"testing"

	statsv5 "github.com/segmentio/stats/v5"
)

// DisableVersionReporting disables the v5 behavior of injecting
// `go_version` and `stats_version` tags,
// which v4 tests don't necessarily expect.
func DisableVersionReporting(tb testing.TB) {
	tb.Helper()

	setGlobal(tb, &statsv5.GoVersionReportingEnabled, false)
}

func setGlobal[X any](tb testing.TB, ptr *X, val X) {
	tb.Helper()

	prev := *ptr
	tb.Cleanup(func() { *ptr = prev })

	*ptr = val
}
