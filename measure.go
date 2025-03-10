package stats

import (
	statsv5 "github.com/segmentio/stats/v5"
)

// Measure behaves like [stats/v5.Measure].
type Measure = statsv5.Measure

// MakeMeasures behaves like [stats/v5.MakeMeasures].
func MakeMeasures(prefix string, value interface{}, tags ...Tag) []Measure {
	return statsv5.MakeMeasures(prefix, value, tags...)
}
