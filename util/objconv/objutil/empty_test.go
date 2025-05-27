package objutil

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestIsEmptyTrue(t *testing.T) {
	tests := []interface{}{
		nil,

		false,

		int(0),
		int8(0),
		int16(0),
		int32(0),
		int64(0),

		uint(0),
		uint8(0),
		uint16(0),
		uint32(0),
		uint64(0),
		uintptr(0),

		float32(0),
		float64(0),

		"",
		[]byte(nil),
		[]int{},
		[...]int{},

		(map[string]int)(nil),
		map[string]int{},

		(*int)(nil),
		unsafe.Pointer(nil),

		(chan struct{})(nil),
		(func())(nil),
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test), func(t *testing.T) {
			if !IsEmpty(test) {
				t.Errorf("%T, %#v should be an empty value", test, test)
			}
		})
	}
}

func TestIsEmptyFalse(t *testing.T) {
	answer := 42

	tests := []interface{}{
		true,

		int(1),
		int8(1),
		int16(1),
		int32(1),
		int64(1),

		uint(1),
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		uintptr(1),

		float32(1),
		float64(1),

		"Hello World!",
		[]byte("Hello World!"),
		[]int{1, 2, 3},
		[...]int{1, 2, 3},

		map[string]int{"answer": 42},

		&answer,
		unsafe.Pointer(&answer),

		make(chan struct{}),
		func() {},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test), func(t *testing.T) {
			if IsEmpty(test) {
				t.Errorf("%T, %#v should not be an empty value", test, test)
			}
		})
	}
}
