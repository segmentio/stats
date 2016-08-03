package stats

import (
	"errors"
	"reflect"
	"testing"
)

func TestAppendError(t *testing.T) {
	err := errors.New("test")

	tests := []struct {
		list error
		err  error
		res  error
	}{
		{
			list: nil,
			err:  nil,
			res:  nil,
		},
		{
			list: err,
			err:  nil,
			res:  err,
		},
		{
			list: nil,
			err:  err,
			res:  err,
		},
		{
			list: err,
			err:  err,
			res:  multiError{err, err},
		},
		{
			list: multiError{err},
			err:  err,
			res:  multiError{err, err},
		},
		{
			list: multiError{err},
			err:  multiError{err},
			res:  multiError{err, err},
		},
	}

	for _, test := range tests {
		if res := appendError(test.list, test.err); !reflect.DeepEqual(res, test.res) {
			t.Errorf("appendError(%v, %v): %#v != %#v", test.list, test.err, test.res, res)
		}
	}
}

func TestMultiError(t *testing.T) {
	tests := []struct {
		err error
		str string
	}{
		{
			err: multiError{},
			str: "",
		},
		{
			err: multiError{errors.New("A")},
			str: "A",
		},
		{
			err: multiError{
				errors.New("A"),
				errors.New("B"),
				errors.New("C"),
			},
			str: "A\nB\nC\n",
		},
	}

	for _, test := range tests {
		if s := test.err.Error(); s != test.str {
			t.Errorf("invalid error string: %#v != %#v", test.str, s)
		}
	}
}
