package stats

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMustValueOf(t *testing.T) {
	tests := []struct {
		name  string
		in    interface{}
		out   interface{}
		panic bool
	}{
		{
			name: "should not panic",
			in:   42,
			out:  ValueOf(42),
		},
		{
			name:  "should panic",
			in:    struct{}{},
			panic: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.panic {
				require.PanicsWithValue(t, "stats.MustValueOf received a value of unsupported type", func() {
					MustValueOf(ValueOf(test.in))
				})
			} else {
				out := MustValueOf(ValueOf(test.in))
				require.EqualValues(t, test.out, out)
			}
		})
	}
}

func TestValueOf(t *testing.T) {
	tests := []struct {
		in  interface{}
		out interface{}
	}{
		{nil, nil},
		{true, true},
		{false, false},
		{int8(1), int64(1)},
		{int8(-1), int64(-1)},
		{int16(1), int64(1)},
		{int16(-1), int64(-1)},
		{int32(1), int64(1)},
		{int32(-1), int64(-1)},
		{int64(1), int64(1)},
		{int64(-1), int64(-1)},
		{uint8(1), uint64(1)},
		{uint16(1), uint64(1)},
		{uint32(1), uint64(1)},
		{uint64(1), uint64(1)},
		{uintptr(1), uint64(1)},
		{float32(0.5), float64(0.5)},
		{float64(0.5), float64(0.5)},
		{time.Second, time.Second},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T(%v)", test.in, test.in), func(t *testing.T) {
			v := ValueOf(test.in).Interface()

			if !reflect.DeepEqual(v, test.out) {
				t.Errorf("bad value: %T(%v)", v, v)
			}
		})
	}
}

func BenchmarkValueOf(b *testing.B) {
	for i := 0; i != b.N; i++ {
		ValueOf(42)
	}
}
