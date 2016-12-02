package iostats

import "io"

// CountReader is an io.Reader that counts how many bytes are read by calls to
// the Read method.
type CountReader struct {
	R io.Reader
	N int
}

// Read satisfies the io.Reader interface.
func (r *CountReader) Read(b []byte) (n int, err error) {
	if n, err = r.R.Read(b); n > 0 {
		r.N += n
	}
	return
}

// CountWriter is an io.Writer that counts how many bytes are written by calls
// to the Write method.
type CountWriter struct {
	W io.Writer
	N int
}

// Write satisfies the io.Writer interface.
func (w *CountWriter) Write(b []byte) (n int, err error) {
	if n, err = w.W.Write(b); n > 0 {
		w.N += n
	}
	return
}

// ReaderFunc makes it possible for function types to be used as io.Reader.
type ReaderFunc func([]byte) (int, error)

// Read calls f(b).
func (f ReaderFunc) Read(b []byte) (int, error) { return f(b) }

// WriterFunc makes it possible for function types to be used as io.Writer.
type WriterFunc func([]byte) (int, error)

// Write calls f(b).
func (f WriterFunc) Write(b []byte) (int, error) { return f(b) }

// CloserFunc makes it possible for function types to be used as io.Closer.
type CloserFunc func() error

// Close calls f.
func (f CloserFunc) Close() error { return f() }
