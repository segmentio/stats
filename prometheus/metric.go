package prometheus

import "time"

type metricType int

const (
	untyped metricType = iota
	counter
	gauge
	histogram
)

func (t metricType) String() string {
	switch t {
	case untyped:
		return "untyped"
	case counter:
		return "counter"
	case gauge:
		return "gauge"
	case histogram:
		return "histogram"
	default:
		return "unknown"
	}
}

type metric struct {
	mtype  metricType
	name   string
	help   string
	value  float64
	time   time.Time
	labels []label
}

type label struct {
	name  string
	value string
}
