package datadog

import (
	"fmt"
	"io"

	"github.com/segmentio/stats"
)

type MetricType string

const (
	Gauge     MetricType = "g"
	Counter   MetricType = "c"
	Histogram MetricType = "h"
)

type Metric struct {
	Name       string
	Value      float64
	Type       MetricType
	SampleRate SampleRate
	Tags       Tags
}

func (m Metric) Format(f fmt.State, _ rune) {
	fmt.Fprintf(f, "%s:%g|%s%v%v\n", m.Name, m.Value, m.Type, m.SampleRate, m.Tags)
}

type Tags stats.Tags

func (tags Tags) Format(f fmt.State, _ rune) {
	if len(tags) != 0 {
		io.WriteString(f, "|#")

		for i, t := range tags {
			if i != 0 {
				io.WriteString(f, ",")
			}
			io.WriteString(f, sanitize(t.Name))
			io.WriteString(f, ":")
			io.WriteString(f, sanitize(t.Value))
		}
	}
}

type SampleRate float64

func (r SampleRate) Format(f fmt.State, _ rune) {
	if r != 1 {
		fmt.Fprintf(f, "|@%g", float64(r))
	}
}
