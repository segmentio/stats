package iostats

import "io"

type CountReader struct {
	R io.Reader
	N int
}

func (r *CountReader) Read(b []byte) (n int, err error) {
	if n, err = r.R.Read(b); n > 0 {
		r.N += n
	}
	return
}

type CountWriter struct {
	W io.Writer
	N int
}

func (w *CountWriter) Write(b []byte) (n int, err error) {
	if n, err = w.W.Write(b); n > 0 {
		w.N += n
	}
	return
}

type ReaderFunc func([]byte) (int, error)

func (f ReaderFunc) Read(b []byte) (int, error) { return f(b) }

type WriterFunc func([]byte) (int, error)

func (f WriterFunc) Write(b []byte) (int, error) { return f(b) }

type CloserFunc func() error

func (f CloserFunc) Close() error { return f() }
