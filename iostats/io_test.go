package iostats

import (
	"bytes"
	"io"
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

		if n, err := c.Read(b); err != nil && err != io.EOF {
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

func TestReaderFunc(t *testing.T) {
	r := ReaderFunc(strings.NewReader("Hello World!").Read)
	b := make([]byte, 20)

	if n, err := r.Read(b); err != nil {
		t.Errorf("ReaderFunc.Read was expected to return no error but got %v", err)
	} else if n != 12 {
		t.Errorf("ReaderFunc.Read was expected to return 12 bytes but got %d", n)
	} else if s := string(b[:n]); s != "Hello World!" {
		t.Errorf("ReaderFunc.Read filled the buffer with invalid content: %#v", s)
	}
}

func TestWriterFunc(t *testing.T) {
	b := &bytes.Buffer{}
	w := WriterFunc(b.Write)

	if n, err := w.Write([]byte("Hello World!")); err != nil {
		t.Errorf("WriterFunc.Write was expected to return no error but got %v", err)
	} else if n != 12 {
		t.Errorf("WriterFunc.Write was expected to return 12 bytes but got %d", n)
	} else if s := b.String(); s != "Hello World!" {
		t.Errorf("WriterFunc.Write filled the buffer with invalid content: %#v", s)
	}
}

func TestCloserFunc(t *testing.T) {
	c := CloserFunc(func() error { return io.EOF })

	if err := c.Close(); err != io.EOF {
		t.Errorf("CloseFunc.Close returned an invalid error, expected EOF but got %v", err)
	}
}
