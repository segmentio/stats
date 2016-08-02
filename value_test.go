package stats

import (
	"testing"
	"time"
)

func TestValues(t *testing.T) {
	tests := []struct {
		object Value
		symbol string
		value  float64
	}{
		{
			object: NewValue("test", 1),
			symbol: "test",
			value:  1,
		},
		{
			object: Count(1),
			symbol: "count",
			value:  1,
		},
		{
			object: Size(1),
			symbol: "size",
			value:  1,
		},
		{
			object: Bytes(1),
			symbol: "bytes",
			value:  1,
		},
		{
			object: Duration(time.Second),
			symbol: "duration",
			value:  1,
		},
	}

	for _, test := range tests {
		if symbol := test.object.Measure(); symbol != test.symbol {
			t.Errorf("%#v: invalid symbol: %#v != %#v", test.object, test.symbol, symbol)
		}

		if value := test.object.Value(); value != test.value {
			t.Errorf("%#v: invalid value: %#v != %#v", test.object, test.value, value)
		}
	}
}

func TestIncrValue(t *testing.T) {
	v0 := NewValue("test", 1)
	v1 := Incr(v0)

	switch x := v1.(type) {
	case Increment:
		if s := x.Measure(); s != v0.Measure() {
			t.Errorf("incr: invalid measure: %#v != %#v", v0.Measure(), s)
		}

		if v := x.Value(); v != v0.Value() {
			t.Errorf("incr: invalid value: %#v != %#v", v0.Value(), v)
		}

	default:
		t.Errorf("incr: invalid type: %T", x)
	}
}

func TestIncrIncrement(t *testing.T) {
	v0 := Incr(NewValue("test", 1))
	v1 := Incr(v0)

	if v0 != v1 {
		t.Errorf("incr: %#v != %#v", v0, v1)
	}
}

func TestDecrValue(t *testing.T) {
	v0 := NewValue("test", 1)
	v1 := Decr(v0)

	switch x := v1.(type) {
	case Decrement:
		if s := x.Measure(); s != v0.Measure() {
			t.Errorf("decr: invalid measure: %#v != %#v", v0.Measure(), s)
		}

		if v := x.Value(); v != v0.Value() {
			t.Errorf("decr: invalid value: %#v != %#v", v0.Value(), v)
		}

	default:
		t.Errorf("decr: invalid type: %T", x)
	}
}

func TestDecrDecrement(t *testing.T) {
	v0 := Decr(NewValue("test", 1))
	v1 := Decr(v0)

	if v0 != v1 {
		t.Errorf("decr: %#v != %#v", v0, v1)
	}
}
