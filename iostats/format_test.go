package iostats

import (
	"testing"
	"unicode"
)

func TestFormatBool(t *testing.T) {
	tests := []struct {
		v bool
		s string
	}{
		{
			v: false,
			s: "false",
		},
		{
			v: true,
			s: "true",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatBool(test.v)); s != test.s {
			t.Errorf("FormatBool: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		v int64
		s string
	}{
		{
			v: 0,
			s: "0",
		},
		{
			v: 42,
			s: "42",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatInt(test.v, 10)); s != test.s {
			t.Errorf("FormatInt: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatUint(t *testing.T) {
	tests := []struct {
		v uint64
		s string
	}{
		{
			v: 0,
			s: "0",
		},
		{
			v: 42,
			s: "42",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatUint(test.v, 10)); s != test.s {
			t.Errorf("FormatUint: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		v float64
		s string
	}{
		{
			v: 0,
			s: "0",
		},
		{
			v: 1.234,
			s: "1.234",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatFloat(test.v, 'g', -1, 64)); s != test.s {
			t.Errorf("FormatFloat: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatByte(t *testing.T) {
	tests := []struct {
		v byte
		s string
	}{
		{
			v: '0',
			s: "0",
		},
		{
			v: 'A',
			s: "A",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatByte(test.v)); s != test.s {
			t.Errorf("FormatByte: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatRune(t *testing.T) {
	tests := []struct {
		v rune
		s string
	}{
		{
			v: '0',
			s: "0",
		},
		{
			v: 'A',
			s: "A",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatRune(test.v)); s != test.s {
			t.Errorf("FormatRune: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		v string
		s string
	}{
		{
			v: "",
			s: "",
		},
		{
			v: "Hello World!",
			s: "Hello World!",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatString(test.v)); s != test.s {
			t.Errorf("FormatString: %#v != %#v", test.s, s)
		}
	}
}

func TestFormatStringFunc(t *testing.T) {
	tests := []struct {
		v string
		s string
	}{
		{
			v: "",
			s: "",
		},
		{
			v: "Hello World!",
			s: "hello world!",
		},
	}

	f := Formatter{}
	defer f.Release()

	for _, test := range tests {
		if s := string(f.FormatStringFunc(test.v, unicode.ToLower)); s != test.s {
			t.Errorf("FormatString: %#v != %#v", test.s, s)
		}
	}
}
