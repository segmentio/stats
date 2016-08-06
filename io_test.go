package stats

import (
	"bytes"
	"strings"
	"testing"
)

func TestCountReader(t *testing.T) {
	tests := []struct {
		s string
	}{
		{
			s: "",
		},
		{
			s: "Hello World!",
		},
	}

	for _, test := range tests {
		c := &CountReader{R: strings.NewReader(test.s)}
		b := make([]byte, len(test.s))

		if n, err := c.Read(b); err != nil {
			t.Error(err)
		} else if n != len(test.s) {
			t.Errorf("invalid byte count returned by the reader: %d != %d", len(test.s), n)
		} else if s := string(b); s != test.s {
			t.Errorf("invalid content returned by the reader: %#v != %#v", test.s, s)
		}
	}
}

func TestCountWriter(t *testing.T) {
	tests := []struct {
		s string
	}{
		{
			s: "",
		},
		{
			s: "Hello World!",
		},
	}

	for _, test := range tests {
		b := &bytes.Buffer{}
		c := &CountWriter{W: b}

		if n, err := c.Write([]byte(test.s)); err != nil {
			t.Error(err)
		} else if n != len(test.s) {
			t.Errorf("invalid byte count returned by the writer: %d != %d", len(test.s), n)
		} else if s := b.String(); s != test.s {
			t.Errorf("invalid content returned by the writer: %#v != %#v", test.s, s)
		}
	}
}
