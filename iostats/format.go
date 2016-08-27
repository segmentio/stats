package iostats

import (
	"strconv"
	"sync"
	"unicode/utf8"
)

// A Formatter exposes a couple of optimized routines for formatting values to
// different representations. It maintains an internal buffer to reuse memory in
// an efficient way.
//
// Formatter buffers are also allocated from memory pools to avoid re-allocation
// of memory buffers as much as possible.
//
// The zero-value of a formatter is a valid value, buffers are lazy-allocated on
// demand when the formatter is used.
//
// Formatters are not safe to use concurrently from multiple goroutines.
type Formatter struct {
	b []byte            // general purpose buffer
	r [utf8.UTFMax]byte // rune buffer
}

// Release puts the formatter's internal buffer pack into the internal memory
// pool.
func (f *Formatter) Release() {
	b := f.b
	f.b = nil
	formatterPool.Put(b)
}

// FormatBool calls strconv.FormatBool using the formatter's internal buffer
// then returns the formatted value.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatBool(v bool) []byte {
	f.b = strconv.AppendBool(f.buffer(), v)
	return f.b
}

// FormatInt calls strconv.FormatInt using the formatter's internal buffer
// then returns the formatted value.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatInt(v int64, base int) []byte {
	f.b = strconv.AppendInt(f.buffer(), v, base)
	return f.b
}

// FormatUint calls strconv.FormatUint using the formatter's internal buffer
// then returns the formatted value.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatUint(v uint64, base int) []byte {
	f.b = strconv.AppendUint(f.buffer(), v, base)
	return f.b
}

// FormatFloat calls strconv.FormatFloat using the formatter's internal
// buffer then returns the formatted value.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatFloat(v float64, format byte, precision int, bitSize int) []byte {
	f.b = strconv.AppendFloat(f.buffer(), v, format, precision, bitSize)
	return f.b
}

// FormatByte converts a byte to a byte slice using the formatter's internal
// memory buffer.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatByte(b byte) []byte {
	f.b = append(f.buffer(), b)
	return f.b
}

// FormatRune converts a rune to a byte slice using the formatter's internal
// memory buffer.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatRune(r rune) []byte {
	n := utf8.EncodeRune(f.r[:], r)
	f.b = append(f.buffer(), f.r[:n]...)
	return f.b
}

// FormatString converts a string to a byte slice using the formatter's internal
// memory buffer.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatString(s string) []byte {
	f.b = append(f.buffer(), s...)
	return f.b
}

// FormatString converts a string to a byte slice using the formatter's internal
// memory buffer, applying t to each rune of the string.
//
// The returned byte slice is value until the next call to a formatter's
// method.
func (f *Formatter) FormatStringFunc(s string, t func(rune) rune) []byte {
	b := f.buffer()

	for _, r := range s {
		n := utf8.EncodeRune(f.r[:], t(r))
		b = append(b, f.r[:n]...)
	}

	f.b = b
	return b
}

func (f *Formatter) buffer() []byte {
	if f.b == nil {
		f.b = formatterPool.Get().([]byte)
	}
	return f.b[:0]
}

const (
	defaultFormatterBufferCapacity = 512
)

var (
	formatterPool = sync.Pool{
		New: func() interface{} { return make([]byte, 0, defaultFormatterBufferCapacity) },
	}
)
