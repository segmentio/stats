package stats

import (
	statsv5 "github.com/segmentio/stats/v5"
)

// Field behaves like [stats/v5.Field].
type Field = statsv5.Field

// MakeField behaves like [stats/v5.MakeField].
func MakeField(name string, value interface{}, ftype FieldType) Field {
	return statsv5.MakeField(name, value, ftype)
}

// FieldType behaves like [stats/v5.FieldType].
type FieldType = statsv5.FieldType

// FieldType constants (see same-named constants in [stats/v5]).
const (
	Counter   = statsv5.Counter
	Gauge     = statsv5.Gauge
	Histogram = statsv5.Histogram
)
