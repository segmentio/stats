package stats

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

type readCloser struct {
	io.Reader
	io.Closer
}

type nopeReadCloser struct{}

func (n nopeReadCloser) Close() error { return nil }

func (n nopeReadCloser) Read(b []byte) (int, error) { return 0, io.EOF }
