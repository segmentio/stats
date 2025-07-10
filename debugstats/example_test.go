package debugstats_test

import (
	"regexp"

	"github.com/segmentio/stats/v5"
	"github.com/segmentio/stats/v5/debugstats"
)

// ExampleClient demonstrates how to register a debugstats.Client so that all
// produced metrics are echoed to stdout.  The example purposefully omits an
// output check because the first column is a time-stamp whose value varies.
func ExampleClient() {
	// Only show metrics whose name contains "foo".
	stats.Register(&debugstats.Client{
		Grep: regexp.MustCompile(`foo`),
	})

	stats.Set("foo_active_users", 123)
	stats.Observe("bar_compression_ratio", 0.37) // <- this one is filtered out

	// Flush to make sure the handler has processed everything before the
	// program exits in short-lived examples or CLI tools.
	stats.Flush()
}
