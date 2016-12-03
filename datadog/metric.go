package datadog

import "github.com/segmentio/stats"

// MetricType is an enumeration providing symbols to represent the different
// metric types upported by datadog.
type MetricType string

const (
	Counter   MetricType = "c"
	Gauge     MetricType = "g"
	Histogram MetricType = "h"
)

// The Metric type is a representation of the metrics supported by datadog.
type Metric struct {
	Type      MetricType      // the metric type
	Name      string          // the metric name
	Value     float64         // the metric value
	Rate      float64         // sample rate, a value between 0 and 1
	Tags      []stats.Tag     // the list of tags set on the metric
	Namespace stats.Namespace // the metric namespace (never populated by parsing operations)
}
