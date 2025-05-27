package objconv

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// Used for testing that the encoder does the right thing with type aliases.
type TBool bool
type TInt int
type TInt8 int8
type TInt16 int16
type TInt32 int32
type TInt64 int64
type TUint uint
type TUint8 uint8
type TUint16 uint16
type TUint32 uint32
type TUint64 uint64
type TUintptr uintptr
type TFloat32 float32
type TFloat64 float64
type TString string
type TBytes []byte

func TestEncoder(t *testing.T) {
	now := time.Now()
	err := errors.New("error")

	tests := [...]struct {
		in  interface{}
		out interface{}
	}{
		// nil
		{nil, nil},

		// booleans
		{true, true},
		{false, false},
		{TBool(true), true},
		{TBool(false), false},

		// int
		{int(1), int64(1)},
		{int8(1), int64(1)},
		{int16(1), int64(1)},
		{int32(1), int64(1)},
		{int64(1), int64(1)},
		{TInt(1), int64(1)},
		{TInt8(1), int64(1)},
		{TInt16(1), int64(1)},
		{TInt32(1), int64(1)},
		{TInt64(1), int64(1)},

		// uint
		{uint(1), uint64(1)},
		{uint8(1), uint64(1)},
		{uint16(1), uint64(1)},
		{uint32(1), uint64(1)},
		{uint64(1), uint64(1)},
		{uintptr(1), uint64(1)},
		{TUint(1), uint64(1)},
		{TUint8(1), uint64(1)},
		{TUint16(1), uint64(1)},
		{TUint32(1), uint64(1)},
		{TUint64(1), uint64(1)},
		{TUintptr(1), uint64(1)},

		// float
		{float32(1), float64(1)},
		{float64(1), float64(1)},
		{TFloat32(1), float64(1)},
		{TFloat64(1), float64(1)},

		// string
		{"Hello World!", "Hello World!"},
		{TString("Hello World!"), "Hello World!"},

		// bytes
		{[]byte("123"), []byte("123")},
		{TBytes("123"), []byte("123")},

		// time
		{now, now},

		// duration
		{time.Second, time.Second},

		// error
		{err, err},

		// array
		{[...]int{1, 2, 3}, []interface{}{int64(1), int64(2), int64(3)}},
		{[]int{1, 2, 3}, []interface{}{int64(1), int64(2), int64(3)}},
		{[]int{}, []interface{}{}},

		// map
		{map[int]int{1: 21, 2: 42}, map[interface{}]interface{}{
			int64(1): int64(21),
			int64(2): int64(42),
		}},
		{map[string]interface{}{"hello": "world"}, map[interface{}]interface{}{
			"hello": "world",
		}},

		// struct
		{struct{}{}, map[interface{}]interface{}{}},
		{struct{ A int }{42}, map[interface{}]interface{}{"A": int64(42)}},

		// struct tags
		{
			in: &struct {
				A bool `objconv:"a"`
				B bool `objconv:"b,omitempty"`
				C bool `objconv:"c,omitzero"`
			}{true, false, false},
			out: map[interface{}]interface{}{
				"a": true,
			},
		},

		// struct tags (json)
		{
			in: &struct {
				A bool `json:"a"`
				B bool `json:"b,omitempty"`
				C bool `json:"c,omitzero"`
			}{true, false, false},
			out: map[interface{}]interface{}{
				"a": true,
				"c": false,
			},
		},

		// list of complex data structures
		{
			in: []map[string]string{
				{"A": "hello", "B": "world"},
				{},
				{"A": "1"},
			},
			out: []interface{}{
				map[interface{}]interface{}{"A": "hello", "B": "world"},
				map[interface{}]interface{}{},
				map[interface{}]interface{}{"A": "1"},
			},
		},

		// nested structs and maps
		{
			in: map[string]struct{ M map[int][]int }{
				"answer": {map[int][]int{1: {1, 2, 3}, 2: nil}},
			},
			out: map[interface{}]interface{}{
				"answer": map[interface{}]interface{}{
					"M": map[interface{}]interface{}{
						int64(1): []interface{}{int64(1), int64(2), int64(3)},
						int64(2): []interface{}{},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test.in), func(t *testing.T) {
			emt := &ValueEmitter{}
			enc := NewEncoder(emt)

			if err := enc.Encode(test.in); err != nil {
				t.Error(err)
			}

			if val := emt.Value(); !reflect.DeepEqual(val, test.out) {
				t.Errorf("%T => %#v", val, val)
			}
		})
	}
}

func BenchmarkEncoder(b *testing.B) {
	tests := [...]interface{}{
		// nil
		nil,

		// bool
		false,
		TBool(false),

		// int
		int(0),
		int8(0),
		int16(0),
		int32(0),
		int64(0),
		TInt(0),
		TInt8(0),
		TInt16(0),
		TInt32(0),
		TInt64(0),

		// uint
		uint(0),
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		uintptr(0),
		TUint(0),
		TUint8(0),
		TUint16(0),
		TUint32(0),
		TUint64(0),
		TUintptr(0),

		// float
		float32(0),
		float64(0),
		TFloat32(0),
		TFloat64(0),

		// string
		"",
		TString(""),

		// bytes
		[]byte(nil),
		TBytes(nil),

		// time
		time.Now(),

		// duration
		time.Second,

		// error
		errors.New("error"),

		// array
		[]int(nil),
		[0]int{},

		// map
		(map[int]int)(nil),
		(map[string]string)(nil),

		// struct
		struct{}{},
		struct{ A int }{},
	}

	enc := NewEncoder(Discard)

	for _, test := range tests {
		b.Run(fmt.Sprintf("%T", test), func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				enc.Encode(test)
			}
		})
	}
}

func TestStreamEncoderFix(t *testing.T) {
	val := &ValueEmitter{}
	enc := NewStreamEncoder(val)

	if err := enc.Open(10); err != nil {
		t.Error(err)
	}

	for i := 0; i != 10; i++ {
		if err := enc.Encode(i); err != nil {
			t.Error(err)
		}
	}

	x1 := []interface{}{
		int64(0),
		int64(1),
		int64(2),
		int64(3),
		int64(4),
		int64(5),
		int64(6),
		int64(7),
		int64(8),
		int64(9),
	}

	x2 := val.Value()

	if !reflect.DeepEqual(x1, x2) {
		t.Error(x1, "!=", x2)
	}
}

func TestStreamEncoderVar(t *testing.T) {
	val := &ValueEmitter{}
	enc := NewStreamEncoder(val)

	for i := 0; i != 10; i++ {
		if err := enc.Encode(i); err != nil {
			t.Error(err)
		}
	}

	if err := enc.Close(); err != nil {
		t.Error(err)
	}

	x1 := []interface{}{
		int64(0),
		int64(1),
		int64(2),
		int64(3),
		int64(4),
		int64(5),
		int64(6),
		int64(7),
		int64(8),
		int64(9),
	}

	x2 := val.Value()

	if !reflect.DeepEqual(x1, x2) {
		t.Error(x1, "!=", x2)
	}
}
