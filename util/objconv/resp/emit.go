package resp

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/stats/v5/util/objconv/objutil"
)

var (
	crlfBytes  = [...]byte{'\r', '\n'}
	nullBytes  = [...]byte{'$', '-', '1', '\r', '\n'}
	trueBytes  = [...]byte{'+', 't', 'r', 'u', 'e', '\r', '\n'}
	falseBytes = [...]byte{'+', 'f', 'a', 'l', 's', 'e', '\r', '\n'}
)

// Emitter implements a RESP emitter that satisfies the objconv.Emitter
// interface.
type Emitter struct {
	w io.Writer

	// This byte slice is used as a local buffer to format values before they
	// are written to the output.
	s []byte

	// This array acts as the initial buffer for s to avoid dynamic memory
	// allocations for the most common use cases.
	a [128]byte

	// This stack is used to cache arrays that are emitted in streaming mode,
	// where the length of the array is not known before outputing all the
	// elements.
	stack []*context

	// sback is used as the initial backing array for the stack slice to avoid
	// dynamic memory allocations for the most common use cases.
	sback [8]*context
}

type context struct {
	b bytes.Buffer // buffer where the array elements are cached
	w io.Writer    // the previous writer where b will be flushed
	n int          // the length of the array as initially set by the encoder
	i int          // the number of elements written to the array
}

func NewEmitter(w io.Writer) *Emitter {
	e := &Emitter{w: w}
	e.s = e.a[:0]
	e.stack = e.sback[:0]
	return e
}

func (e *Emitter) Reset(w io.Writer) {
	e.w = w

	if e.stack == nil {
		e.stack = e.stack[:0]
	} else {
		e.stack = e.stack[:0]
	}

	if e.s == nil {
		e.s = e.a[:0]
	} else {
		e.s = e.s[:0]
	}
}

func (e *Emitter) EmitNil() (err error) {
	_, err = e.w.Write(nullBytes[:])
	return
}

func (e *Emitter) EmitBool(v bool) (err error) {
	if v {
		_, err = e.w.Write(trueBytes[:])
	} else {
		_, err = e.w.Write(falseBytes[:])
	}
	return
}

func (e *Emitter) EmitInt(v int64, _ int) (err error) {
	s := e.s[:0]

	s = append(s, ':')
	s = appendInt(s, v)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitUint(v uint64, _ int) (err error) {
	if v > objutil.Int64Max {
		return fmt.Errorf("objconv/resp: %d overflows the maximum integer value of %d", v, objutil.Int64Max)
	}

	s := e.s[:0]

	s = append(s, ':')
	s = appendUint(s, v)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitFloat(v float64, bitSize int) (err error) {
	s := e.s[:0]

	s = append(s, '+')
	s = appendFloat(s, v, bitSize)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitString(v string) (err error) {
	s := e.s[:0]

	if indexCRLF(v) < 0 {
		s = append(s, '+')
		s = append(s, v...)
		s = appendCRLF(s)
	} else {
		s = append(s, '$')
		s = appendUint(s, uint64(len(v)))
		s = appendCRLF(s)
		s = append(s, v...)
		s = appendCRLF(s)
	}

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitBytes(v []byte) (err error) {
	s := e.s[:0]

	s = append(s, '$')
	s = appendUint(s, uint64(len(v)))
	s = appendCRLF(s)

	if (len(v) + 2) <= (cap(s) - len(s)) { // if it fits in the buffer
		s = append(s, v...)
		s = appendCRLF(s)
		e.s = s[:0]

		_, err = e.w.Write(s)
		return
	}

	e.s = s[:0]

	if _, err = e.w.Write(s); err != nil {
		return
	}

	if _, err = e.w.Write(v); err != nil {
		return
	}

	_, err = e.w.Write(crlfBytes[:])
	return
}

func (e *Emitter) EmitTime(v time.Time) (err error) {
	s := e.s[:0]

	s = append(s, '+')
	s = v.AppendFormat(s, time.RFC3339Nano)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitDuration(v time.Duration) (err error) {
	s := e.s[:0]

	s = append(s, '+')
	s = objutil.AppendDuration(s, v)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitError(v error) (err error) {
	x := v.Error()
	s := e.s[:0]

	if i := indexCRLF(x); i >= 0 {
		x = x[:i] // only keep the first line
	}

	s = append(s, '-')
	s = append(s, x...)
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func (e *Emitter) EmitArrayBegin(n int) (err error) {
	var c *context

	if n < 0 {
		c = contextPool.Get().(*context)
		c.b.Truncate(0)
		c.n = 0
		c.w = e.w
		e.w = &c.b
	} else {
		err = e.emitArray(n)
	}

	e.stack = append(e.stack, c)
	return
}

func (e *Emitter) EmitArrayEnd() (err error) {
	i := len(e.stack) - 1
	c := e.stack[i]
	e.stack = e.stack[:i]

	if c != nil {
		e.w = c.w

		if c.b.Len() != 0 {
			c.n++
		}

		if err = e.emitArray(c.n); err == nil {
			_, err = c.b.WriteTo(c.w)
		}

		contextPool.Put(c)
	}

	return
}

func (e *Emitter) EmitArrayNext() (err error) {
	if c := e.stack[len(e.stack)-1]; c != nil {
		c.n++
	}
	return
}

func (e *Emitter) EmitMapBegin(n int) (err error) {
	return e.emitArray(n + n)
}

func (e *Emitter) EmitMapEnd() (err error) {
	return
}

func (e *Emitter) EmitMapValue() (err error) {
	return
}

func (e *Emitter) EmitMapNext() (err error) {
	return
}

func (e *Emitter) emitArray(n int) (err error) {
	s := e.s[:0]

	s = append(s, '*')
	s = appendUint(s, uint64(n))
	s = appendCRLF(s)

	e.s = s[:0]
	_, err = e.w.Write(s)
	return
}

func appendInt(b []byte, v int64) []byte {
	return strconv.AppendInt(b, v, 10)
}

func appendUint(b []byte, v uint64) []byte {
	return strconv.AppendUint(b, v, 10)
}

func appendFloat(b []byte, v float64, bitSize int) []byte {
	return strconv.AppendFloat(b, v, 'g', -1, bitSize)
}

func appendCRLF(b []byte) []byte {
	return append(b, '\r', '\n')
}

func indexCRLF(s string) int {
	for i, n := 0, len(s); i != n; i++ {
		j := strings.IndexByte(s[i:], '\r')

		if j < 0 {
			break
		}

		if j++; j == n {
			break
		}

		if s[j] == '\n' {
			return j - 1
		}
	}
	return -1
}

var contextPool = sync.Pool{
	New: func() interface{} { return &context{} },
}

// ClientEmitter is the implementation of a RESP emitter suitable to be used for
// encoding redis client requests.
type ClientEmitter struct {
	Emitter
}

func NewClientEmitter(w io.Writer) *ClientEmitter {
	e := &ClientEmitter{}
	e.w = w
	e.s = e.a[:0]
	e.stack = e.sback[:0]
	return e
}

func (e *ClientEmitter) EmitBool(v bool) error {
	var x int64

	if v {
		x = 1
	}

	return e.EmitInt(x, 64)
}

func (e *ClientEmitter) EmitInt(v int64, _ int) error {
	a := [64]byte{}
	b := appendInt(a[:0], v)
	return e.EmitBytes(b)
}

func (e *ClientEmitter) EmitUint(v uint64, _ int) error {
	a := [64]byte{}
	b := appendUint(a[:0], v)
	return e.EmitBytes(b)
}

func (e *ClientEmitter) EmitFloat(v float64, bitSize int) error {
	a := [64]byte{}
	b := appendFloat(a[:0], v, bitSize)
	return e.EmitBytes(b)
}

func (e *ClientEmitter) EmitString(v string) (err error) {
	s := e.s[:0]

	s = append(s, '$')
	s = appendUint(s, uint64(len(v)))
	s = appendCRLF(s)
	s = append(s, v...)
	s = appendCRLF(s)

	e.s = s[:0]

	_, err = e.w.Write(s)
	return
}

func (e *ClientEmitter) EmitTime(v time.Time) error {
	return e.EmitInt(v.Unix(), 64)
}

func (e *ClientEmitter) EmitDuration(v time.Duration) error {
	return e.EmitFloat(v.Seconds(), 64)
}

func (e *ClientEmitter) EmitError(v error) error {
	return e.EmitString(v.Error())
}
