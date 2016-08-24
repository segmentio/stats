package linux

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

func TestConvertPanicToError(t *testing.T) {
	tests := []struct {
		v interface{}
		e error
	}{
		{
			v: nil,
			e: nil,
		},
		{
			v: io.EOF,
			e: io.EOF,
		},
		{
			v: "Hello World!",
			e: errors.New("Hello World!"),
		},
	}

	for _, test := range tests {
		if err := convertPanicToError(test.v); !reflect.DeepEqual(err, test.e) {
			t.Errorf("bad error from panic: %v != %v", test.e, err)
		}
	}
}

func TestCheck(t *testing.T) {
	err := io.EOF

	defer func() {
		if x := recover(); x != err {
			t.Error("invalid panic:", x)
		}
	}()

	check(err)
}
