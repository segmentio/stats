package statsd

import "fmt"

type MetricType string

const (
	Gauge     MetricType = "g"
	Counter   MetricType = "c"
	Histogram MetricType = "h"
	Timer     MetricType = "ms"
	Meter     MetricType = "m"
)

type Metric struct {
	Name   string
	Value  int64
	Type   MetricType
	Sample Sample
}

func (m Metric) Format(f fmt.State, _ rune) {
	fmt.Fprintf(f, "%s:%d|%s%v\n", m.Name, m.Value, m.Type, m.Sample)
}

type Sample float64

func (r Sample) Format(f fmt.State, _ rune) {
	if r != 0 && r != 1 {
		fmt.Fprintf(f, "|@%g", float64(r))
	}
}
