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
		if symbol := test.object.Symbol(); symbol != test.symbol {
			t.Errorf("%#v: invalid symbol: %#v != %#v", test.object, test.symbol, symbol)
		}

		if value := test.object.Value(); value != test.value {
			t.Errorf("%#v: invalid value: %#v != %#v", test.object, test.value, value)
		}
	}
}
