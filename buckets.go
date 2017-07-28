package stats

// Key is a type used to uniquely identify metrics.
type Key struct {
	Measure string
	Field   string
}

// Buckets is a registry where histogram buckets are placed. Some metric
// collection backends need to have histogram buckets defined by the program
// (like Prometheus), a common pattern is to use the init function of a package
// to register buckets for the various histograms that it produces.
var Buckets = map[Key][]Value{}
