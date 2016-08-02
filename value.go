package stats

import "time"

type Value interface {
	Symbol() string

	Value() float64
}

func NewValue(sym string, val float64) Value {
	return value{symbol: sym, value: val}
}

type value struct {
	symbol string
	value  float64
}

func (v value) Symbol() string { return v.symbol }
func (v value) Value() float64 { return v.value }

type Count uint64

func (v Count) Symbol() string { return "count" }
func (v Count) Value() float64 { return float64(v) }

type Size uint64

func (v Size) Symbol() string { return "size" }
func (v Size) Value() float64 { return float64(v) }

type Bytes uint64

func (v Bytes) Symbol() string { return "bytes" }
func (v Bytes) Value() float64 { return float64(v) }

type Duration time.Duration

func (v Duration) Symbol() string { return "duration" }
func (v Duration) Value() float64 { return time.Duration(v).Seconds() }
