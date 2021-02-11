package stats

import (
	"math"
	"strconv"
	"time"
)

// Value is a wrapper type which is used to encapsulate underlying types (nil, bool, int, uintptr, float)
// in a single pseudo-generic type.
type Value struct {
	typ  Type
	pad  int32
	bits uint64
}

// MustValueOf asserts that v's underlying Type is valid, otherwise it panics.
func MustValueOf(v Value) Value {
	if v.Type() == Invalid {
		panic("stats.MustValueOf received a value of unsupported type")
	}
	return v
}
// ValueOf inspects v's underlying type and returns a Value which encapsulates this type.
// If the underlying type of v is not supported by Value's encapsulation its Type() will return stats.Invalid
func ValueOf(v interface{}) Value {
	switch x := v.(type) {
	case Value:
		return x
	case nil:
		return Value{}
	case bool:
		return boolValue(x)
	case int:
		return intValue(x)
	case int8:
		return int8Value(x)
	case int16:
		return int16Value(x)
	case int32:
		return int32Value(x)
	case int64:
		return int64Value(x)
	case uint:
		return uintValue(x)
	case uint8:
		return uint8Value(x)
	case uint16:
		return uint16Value(x)
	case uint32:
		return uint32Value(x)
	case uint64:
		return uint64Value(x)
	case uintptr:
		return uintptrValue(x)
	case float32:
		return float32Value(x)
	case float64:
		return float64Value(x)
	case time.Duration:
		return durationValue(x)
	default:
		return Value{typ: Invalid}
	}
}

func boolValue(v bool) Value {
	return Value{typ: Bool, bits: boolBits(v)}
}

func intValue(v int) Value {
	return int64Value(int64(v))
}

func int8Value(v int8) Value {
	return int64Value(int64(v))
}

func int16Value(v int16) Value {
	return int64Value(int64(v))
}

func int32Value(v int32) Value {
	return int64Value(int64(v))
}

func int64Value(v int64) Value {
	return Value{typ: Int, bits: uint64(v)}
}

func uintValue(v uint) Value {
	return uint64Value(uint64(v))
}

func uint8Value(v uint8) Value {
	return uint64Value(uint64(v))
}

func uint16Value(v uint16) Value {
	return uint64Value(uint64(v))
}

func uint32Value(v uint32) Value {
	return uint64Value(uint64(v))
}

func uint64Value(v uint64) Value {
	return Value{typ: Uint, bits: v}
}

func uintptrValue(v uintptr) Value {
	return uint64Value(uint64(v))
}

func float32Value(v float32) Value {
	return float64Value(float64(v))
}

func float64Value(v float64) Value {
	return Value{typ: Float, bits: math.Float64bits(v)}
}

func durationValue(v time.Duration) Value {
	return Value{typ: Duration, bits: uint64(v)}
}
// Type returns the Type of this value.
func (v Value) Type() Type {
	return v.typ
}
// Bool returns a bool if the underlying data for this value is zero.
func (v Value) Bool() bool {
	return v.bits != 0
}
// Int returns an new int64 representation of this Value.
func (v Value) Int() int64 {
	return int64(v.bits)
}
// Uint returns a uint64 representation of this Value.
func (v Value) Uint() uint64 {
	return v.bits
}
// Float returns a new float64 representation of this Value.
func (v Value) Float() float64 {
	return math.Float64frombits(v.bits)
}
// Duration returns a new time.Duration representation of this Value.
func (v Value) Duration() time.Duration {
	return time.Duration(v.bits)
}
// Interface returns an new interface{} representation of this value.
// However, if the underlying Type is unsupported it panics.
func (v Value) Interface() interface{} {
	switch v.Type() {
	case Null:
		return nil
	case Bool:
		return v.Bool()
	case Int:
		return v.Int()
	case Uint:
		return v.Uint()
	case Float:
		return v.Float()
	case Duration:
		return v.Duration()
	default:
		panic("unknown type found in a stats.Value")
	}
}
// String returns a string representation of the underling value.
func (v Value) String() string {
	switch v.Type() {
	case Null:
		return "<nil>"
	case Bool:
		return strconv.FormatBool(v.Bool())
	case Int:
		return strconv.FormatInt(v.Int(), 10)
	case Uint:
		return strconv.FormatUint(v.Uint(), 10)
	case Float:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case Duration:
		return v.Duration().String()
	default:
		return "<unknown>"
	}
}
// Type is an int32 type alias used to denote a values underlying type.
type Type int32

// Underlying Types.
const (
	Null Type = iota
	Bool
	Int
	Uint
	Float
	Duration
	Invalid
)
// String returns the string representation of a type.
func (t Type) String() string {
	switch t {
	case Null:
		return "<nil>"
	case Bool:
		return "bool"
	case Int:
		return "int64"
	case Uint:
		return "uint64"
	case Float:
		return "float64"
	case Duration:
		return "time.Duration"
	default:
		return "<unknown>"
	}
}
// GoString implements the GoStringer interface.
func (t Type) GoString() string {
	switch t {
	case Null:
		return "stats.Null"
	case Bool:
		return "stats.Bool"
	case Int:
		return "stats.Int"
	case Uint:
		return "stats.Uint"
	case Float:
		return "stats.Float"
	case Duration:
		return "stats.Duration"
	default:
		return "stats.Type(" + strconv.Itoa(int(t)) + ")"
	}
}

func boolBits(v bool) uint64 {
	if v {
		return 1
	}
	return 0
}
