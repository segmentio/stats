package stats

import (
	"sync"
	"time"
)

// MetricType is an enumeration representing the type of a metric.
type MetricType int

const (
	// CounterType is the constant representing counter metrics.
	CounterType MetricType = iota

	// GaugeType is the constant representing gauge metrics.
	GaugeType

	// HistogramType is the constant representing histogram metrics.
	HistogramType
)

// String satisfies the fmt.Stringer interface.
func (t MetricType) String() string {
	switch t {
	case CounterType:
		return "counter"
	case GaugeType:
		return "gauge"
	case HistogramType:
		return "histogram"
	default:
		return "unknown"
	}
}

// Metric is a universal representation of the state of a metric.
//
// No operations are available on this data type, instead it carries the state
// of a metric a single metric when querying the state of a stats engine.
type Metric struct {
	// Type is a constant representing the type of the metric, which is one of
	// the constants defined by the MetricType enumeration.
	Type MetricType

	// Namespace in which the metric was generated.
	Namespace string

	// Name is the name of the metric as defined by the program.
	Name string

	// Tags is the list of tags set on the metric.
	Tags []Tag

	// Value is the value reported by the metric, for counters this is the value
	// by which the counter is incremented.
	Value float64

	// Time is unused for now, reserved for future extensions.
	Time time.Time
}

// metricPool is used as an internal store to cache metric objects.
var metricPool = sync.Pool{
	New: func() interface{} {
		return &Metric{
			Tags: make([]Tag, 0, 8),
		}
	},
}
