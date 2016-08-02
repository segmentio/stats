package stats

import "time"

type Value interface {
	Measure() string

	Type() string

	Value() float64
}

func NewValue(sym string, val float64) Value {
	return value{symbol: sym, value: val}
}

func Incr(value Value) Value {
	switch v := value.(type) {
	case Increment:
		return v
	}
	return Increment{value: value}
}

func Decr(value Value) Value {
	switch v := value.(type) {
	case Decrement:
		return v
	}
	return Decrement{value: value}
}

type Increment struct {
	value Value
}

func (i Increment) Measure() string { return i.value.Measure() }
func (i Increment) Type() string    { return "add" }
func (i Increment) Value() float64  { return i.value.Value() }

type Decrement struct {
	value Value
}

func (d Decrement) Measure() string { return d.value.Measure() }
func (d Decrement) Type() string    { return "sub" }
func (d Decrement) Value() float64  { return d.value.Value() }

type Count uint64

func (v Count) Measure() string { return "count" }
func (v Count) Type() string    { return "set" }
func (v Count) Value() float64  { return float64(v) }

type Size uint64

func (v Size) Measure() string { return "size" }
func (v Size) Type() string    { return "set" }
func (v Size) Value() float64  { return float64(v) }

type Bytes uint64

func (v Bytes) Measure() string { return "bytes" }
func (v Bytes) Type() string    { return "set" }
func (v Bytes) Value() float64  { return float64(v) }

type Duration time.Duration

func (v Duration) Measure() string { return "duration" }
func (v Duration) Type() string    { return "set" }
func (v Duration) Value() float64  { return time.Duration(v).Seconds() }

type value struct {
	symbol string
	value  float64
}

func (v value) Measure() string { return v.symbol }
func (v value) Type() string    { return "set" }
func (v value) Value() float64  { return v.value }
