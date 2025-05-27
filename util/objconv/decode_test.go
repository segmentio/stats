package objconv

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestDecoderDecodeType(t *testing.T) {
	date := time.Date(2016, 12, 12, 01, 01, 01, 0, time.UTC)
	err := errors.New("error")

	tests := [...]struct {
		in  interface{}
		out interface{}
	}{
		// type -> nil (discard)
		{nil, nil},
		{true, nil},
		{int(1), nil},
		{uint(1), nil},
		{"A", nil},
		{[]byte("A"), nil},
		{err, nil},
		{date, nil},
		{time.Second, nil},
		{[]int{1, 2, 3}, nil},
		{map[string]int{"answer": 42}, nil},
		{struct{ A, B, C int }{1, 2, 3}, nil},

		// type -> string (conversion)
		{true, "true"},
		{false, "false"},
		{int(42), "42"},
		{uint(42), "42"},
		{float64(0.5), "0.5"},
		{date, "2016-12-12T01:01:01Z"},
		{42 * time.Second, "42s"},
		{err, "error"},

		// nil -> bool
		{nil, false},

		// nil -> int
		{nil, int(0)},
		{nil, int8(0)},
		{nil, int16(0)},
		{nil, int32(0)},
		{nil, int64(0)},

		// nil -> uint
		{nil, uint(0)},
		{nil, uint8(0)},
		{nil, uint16(0)},
		{nil, uint32(0)},
		{nil, uint64(0)},
		{nil, uintptr(0)},

		// nil -> float
		{nil, float32(0)},
		{nil, float64(0)},

		// nil -> string
		{nil, ""},

		// nil -> bytes
		{nil, []byte(nil)},

		// nil -> time
		{nil, time.Time{}},

		// nil -> duration
		{nil, time.Duration(0)},

		// nil -> array
		{nil, [...]int{}},
		{nil, [...]int{0, 0, 0}},

		// nil -> slice
		{nil, []int(nil)},

		// nil -> map
		{nil, (map[int]int)(nil)},

		// nil -> struct
		{nil, struct{}{}},
		{nil, struct{ A int }{}},

		// nil -> ptr
		{nil, (*int)(nil)},

		// bool -> bool
		{false, false},
		{true, true},

		// int -> int
		{int64(1), int(1)},
		{int64(1), int8(1)},
		{int64(1), int16(1)},
		{int64(1), int32(1)},
		{int64(1), int64(1)},

		// int -> uint
		{int64(1), uint(1)},
		{int64(1), uint8(1)},
		{int64(1), uint16(1)},
		{int64(1), uint32(1)},
		{int64(1), uint64(1)},

		// int -> float
		{int64(1), float32(1)},
		{int64(1), float64(1)},

		// uint -> uint
		{uint64(1), uint(1)},
		{uint64(1), uint8(1)},
		{uint64(1), uint16(1)},
		{uint64(1), uint32(1)},
		{uint64(1), uint64(1)},
		{uint64(1), uintptr(1)},

		// uint -> int
		{uint64(1), int(1)},
		{uint64(1), int8(1)},
		{uint64(1), int16(1)},
		{uint64(1), int32(1)},
		{uint64(1), int64(1)},

		// uint -> float
		{uint64(1), float32(1)},
		{uint64(1), float64(1)},

		// float -> float
		{float64(1), float32(1)},
		{float64(1), float64(1)},

		// string -> string
		{"Hello World!", "Hello World!"},

		// string -> bytes
		{"Hello World!", []byte("Hello World!")},

		// string -> int
		{"-42", -42},

		// string -> uint
		{"42", uint(42)},

		// string -> float
		{"42.2", 42.2},

		// string -> time
		{"2016-12-12T01:01:01.000Z", date},

		// string -> duration
		{"1s", time.Second},

		// string -> error
		{"error", err},

		// bytes -> bytes
		{[]byte("Hello World!"), []byte("Hello World!")},

		// bytes -> string
		{[]byte("Hello World!"), "Hello World!"},

		// bytes -> int
		{[]byte("-42"), -42},

		// bytes -> uint
		{[]byte("42"), uint(42)},

		// bytes -> float
		{[]byte("42.42"), 42.42},

		// bytes -> time
		{[]byte("2016-12-12T01:01:01.000Z"), date},

		// bytes -> duration
		{[]byte("1s"), time.Second},

		// bytes -> error
		{[]byte("error"), err},

		// time -> time
		{date, date},

		// duration -> duration
		{time.Second, time.Second},

		// error -> error
		{err, err},

		// array -> array
		{[...]int{}, [...]int{}},
		{[...]int{1, 2, 3}, [...]int{1, 2, 3}},

		// slice -> slice
		{[]int{}, []int{}},
		{[]int{1, 2, 3}, []int{1, 2, 3}},

		// map -> map
		{map[int]int{}, map[int]int{}},
		{map[int]int{1: 21, 2: 42}, map[int]int{1: 21, 2: 42}},
		{map[int]map[int]int{}, map[int]map[int]int{}},
		{map[int]map[int]int{1: {2: 3}}, map[int]map[int]int{1: {2: 3}}},

		// map -> struct
		{map[string]int{}, struct{}{}},
		{map[string]int{"A": 42}, struct{ A int }{42}},
		{map[string]interface{}{"A": 1, "B": nil}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": true}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": int(0)}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": uint(0)}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": float64(0)}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": ""}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": []byte(nil)}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": []int{1, 2, 3}}, struct{ A int }{1}},
		{map[string]interface{}{"A": 1, "B": map[int]int{1: 1, 2: 2, 3: 3}}, struct{ A int }{1}},

		// struct -> map
		{struct{}{}, map[string]interface{}{}},
		{struct{ A int }{42}, map[string]interface{}{"A": int64(42)}},
		{struct{}{}, map[interface{}]interface{}{}},
		{struct{ A int }{42}, map[interface{}]interface{}{"A": int64(42)}},
		{struct{}{}, map[string]string{}},
		{struct{ A string }{"42"}, map[string]string{"A": "42"}},

		// struct -> struct
		{struct{}{}, struct{}{}},
		{struct{ A int }{42}, struct{ A int }{42}},
		{struct{ A, B, C int }{1, 2, 3}, struct{ A, B, C int }{1, 2, 3}},

		// struct -> ptr
		{struct{ A int }{42}, &struct{ A int }{42}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T->%T", test.in, test.out), func(t *testing.T) {
			var dec = NewDecoder(NewValueParser(test.in))
			var val reflect.Value
			var ptr interface{}

			if test.out != nil {
				val = reflect.New(reflect.TypeOf(test.out))
				ptr = val.Interface()
			}

			if err := dec.Decode(ptr); err != nil {
				t.Error(err)
			}

			if test.out != nil {
				if v := val.Elem().Interface(); !reflect.DeepEqual(v, test.out) {
					t.Errorf("%T => %#v != %v", v, v, test.out)
				}
			}
		})
	}
}

func TestDecoderDecodeToEmptyInterface(t *testing.T) {
	tests := []interface{}{
		// nil -> interface{}
		nil,

		// bool -> interface{}
		true,
		false,

		// int -> interface{}
		int64(0),
		int64(1),

		// uint -> interface{}
		uint64(0),
		uint64(1),

		// float -> interface{}
		float64(0),
		float64(1),

		// string -> interface{}
		"",
		"Hello World!",

		// bytes -> interface{}
		[]byte(""),
		[]byte("Hello World!"),

		// time -> interface{}
		time.Now(),

		// duration -> interface{}
		1 * time.Second,

		// error -> interface{}
		errors.New("error"),

		// slice -> interface{}
		[]interface{}{},
		[]interface{}{nil, true, false, int64(0), uint64(0), float64(0), "Hello World"},

		// map -> interface{}
		map[interface{}]interface{}{},
		map[interface{}]interface{}{"Hello": "World!"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T->interface{}", test), func(t *testing.T) {
			var dec = NewDecoder(NewValueParser(test))
			var val interface{}

			if err := dec.Decode(&val); err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(test, val) {
				t.Errorf("%T => %#v != %#v", val, val, test)
			}
		})
	}
}

func TestStreamDecoder(t *testing.T) {
	tests := [][]interface{}{
		{},
		{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test), func(t *testing.T) {
			val := NewValueParser(test)
			dec := NewStreamDecoder(val)

			var v interface{}
			var i = int64(0)

			if n := dec.Len(); n != len(test) {
				t.Error("invalid length returned by the stream decoder:", n)
			}

			for dec.Decode(&v) == nil {
				if !reflect.DeepEqual(v, i) {
					t.Error(v, "!=", test)
				}
				i++
			}

			if int(i) != len(test) {
				t.Error(i)
			}

			if err := dec.Err(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestStreamRencode(t *testing.T) {
	tests := []interface{}{
		nil,
		true,
		false,
		int64(1),
		uint64(1),
		float64(1),
		"Hello World!",
		map[interface{}]interface{}{"hello": "world"},
		[]interface{}{
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
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test), func(t *testing.T) {
			in := NewValueParser(test)
			out := NewValueEmitter()

			dec := NewStreamDecoder(in)
			enc, err := dec.Encoder(out)

			if err != nil {
				t.Error(err)
				return
			}

			var v interface{}

			for dec.Decode(&v) == nil {
				if err := enc.Encode(v); err != nil {
					t.Error(err)
				}
				v = nil
			}

			if err := dec.Err(); err != nil {
				t.Error(err)
			}

			if err := enc.Close(); err != nil {
				t.Error(err)
			}

			if v = out.Value(); !reflect.DeepEqual(v, test) {
				t.Error(v, "!=", test)
			}
		})
	}
}
