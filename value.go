package stats

import (
	statsv5 "github.com/segmentio/stats/v5"
)

// Value behaves like [stats/v5.Value].
type Value = statsv5.Value

// MustValueOf behaves like [stats/v5.MustValueOf].
func MustValueOf(v Value) Value {
	return statsv5.MustValueOf(v)
}

// ValueOf behaves like [stats/v5.ValueOf].
func ValueOf(v interface{}) Value {
	return statsv5.ValueOf(v)
}

// Type behaves like [stats/v5.Type].
type Type = statsv5.Type

// Type constants. See same-named constants in [stats/v5].
const (
	Null     = statsv5.Null
	Bool     = statsv5.Bool
	Int      = statsv5.Int
	Uint     = statsv5.Uint
	Float    = statsv5.Float
	Duration = statsv5.Duration
	Invalid  = statsv5.Invalid
)
