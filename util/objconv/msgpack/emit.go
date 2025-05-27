package msgpack

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/segmentio/stats/v5/util/objconv/objutil"
)

// Emitter implements a MessagePack emitter that satisfies the objconv.Emitter
// interface.
type Emitter struct {
	w io.Writer
	b [240]byte

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
	n int          // the number of elements written to the array
}

func NewEmitter(w io.Writer) *Emitter {
	e := &Emitter{w: w}
	e.stack = e.sback[:0]
	return e
}

func (e *Emitter) Reset(w io.Writer) {
	e.w = w
	e.stack = e.stack[:0]
}

func (e *Emitter) EmitNil() (err error) {
	e.b[0] = Nil
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitBool(v bool) (err error) {
	if v {
		e.b[0] = True
	} else {
		e.b[0] = False
	}
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitInt(v int64, _ int) (err error) {
	n := 0

	if v >= 0 {
		switch {
		case v <= objutil.Int8Max:
			e.b[0] = byte(v) | PositiveFixintTag
			n = 1

		case v <= objutil.Int16Max:
			e.b[0] = Int16
			putUint16(e.b[1:], uint16(v))
			n = 3

		case v <= objutil.Int32Max:
			e.b[0] = Int32
			putUint32(e.b[1:], uint32(v))
			n = 5

		default:
			e.b[0] = Int64
			putUint64(e.b[1:], uint64(v))
			n = 9
		}

	} else {
		switch {
		case v >= -31:
			e.b[0] = byte(v) | NegativeFixintTag
			n = 1

		case v >= objutil.Int8Min:
			e.b[0] = Int8
			e.b[1] = byte(v)
			n = 2

		case v >= objutil.Int16Min:
			e.b[0] = Int16
			putUint16(e.b[1:], uint16(v))
			n = 3

		case v >= objutil.Int32Min:
			e.b[0] = Int32
			putUint32(e.b[1:], uint32(v))
			n = 5

		default:
			e.b[0] = Int64
			putUint64(e.b[1:], uint64(v))
			n = 9
		}
	}

	_, err = e.w.Write(e.b[:n])
	return
}

func (e *Emitter) EmitUint(v uint64, _ int) (err error) {
	n := 0

	switch {
	case v <= objutil.Uint8Max:
		e.b[0] = Uint8
		e.b[1] = byte(v)
		n = 2

	case v <= objutil.Uint16Max:
		e.b[0] = Uint16
		putUint16(e.b[1:], uint16(v))
		n = 3

	case v <= objutil.Uint32Max:
		e.b[0] = Uint32
		putUint32(e.b[1:], uint32(v))
		n = 5

	default:
		e.b[0] = Uint64
		putUint64(e.b[1:], v)
		n = 9
	}

	_, err = e.w.Write(e.b[:n])
	return
}

func (e *Emitter) EmitFloat(v float64, bitSize int) (err error) {
	switch bitSize {
	case 32:
		e.b[0] = Float32
		putUint32(e.b[1:], math.Float32bits(float32(v)))
		_, err = e.w.Write(e.b[:5])
	default:
		e.b[0] = Float64
		putUint64(e.b[1:], math.Float64bits(v))
		_, err = e.w.Write(e.b[:9])
	}
	return
}

func (e *Emitter) EmitString(v string) (err error) {
	n := len(v)

	switch {
	case n <= 31:
		e.b[0] = byte(n) | FixstrTag
		n = 1

	case n <= objutil.Uint8Max:
		e.b[0] = Str8
		e.b[1] = byte(n)
		n = 2

	case n <= objutil.Uint16Max:
		e.b[0] = Str16
		putUint16(e.b[1:], uint16(n))
		n = 3

	case n <= objutil.Uint32Max:
		e.b[0] = Str32
		putUint32(e.b[1:], uint32(n))
		n = 5

	default:
		err = fmt.Errorf("objconv/msgpack: string of length %d is too long to be encoded", n)
		return
	}

	for {
		n1 := len(v)
		n2 := len(e.b[n:])

		if n1 > n2 {
			n1 = n2
		}

		copy(e.b[n:], v[:n1])

		if _, err = e.w.Write(e.b[:n+n1]); err != nil {
			return
		}

		v = v[n1:]
		n = 0

		if len(v) == 0 {
			return
		}
	}
}

func (e *Emitter) EmitBytes(v []byte) (err error) {
	n := len(v)

	switch {
	case n <= objutil.Uint8Max:
		e.b[0] = Bin8
		e.b[1] = byte(n)
		n = 2

	case n <= objutil.Uint16Max:
		e.b[0] = Bin16
		putUint16(e.b[1:], uint16(n))
		n = 3

	case n <= objutil.Uint32Max:
		e.b[0] = Bin32
		putUint32(e.b[1:], uint32(n))
		n = 5

	default:
		err = fmt.Errorf("objconv/msgpack: byte slice of length %d is too long to be encoded", n)
		return
	}

	if _, err = e.w.Write(e.b[:n]); err != nil {
		return
	}

	_, err = e.w.Write(v)
	return
}

func (e *Emitter) EmitTime(v time.Time) (err error) {
	const int34Max = 17179869183

	x := ExtTime
	n := 0
	s := v.Unix()
	ns := v.Nanosecond()

	if ns == 0 && s >= 0 && s <= objutil.Uint32Max {
		e.b[0] = Fixext4
		e.b[1] = byte(x)
		putUint32(e.b[2:], uint32(s))
		n = 6
	} else if s >= 0 && s <= int34Max {
		e.b[0] = Fixext8
		e.b[1] = byte(x)
		putUint64(e.b[2:], uint64(s)|(uint64(ns)<<34))
		n = 10
	} else {
		e.b[0] = Ext8
		e.b[1] = 12
		e.b[2] = byte(x)
		putUint32(e.b[3:], uint32(ns))
		putUint64(e.b[7:], uint64(s))
		n = 15
	}

	_, err = e.w.Write(e.b[:n])
	return
}

func (e *Emitter) EmitDuration(v time.Duration) (err error) {
	return e.EmitString(string(objutil.AppendDuration(e.b[:0], v)))
}

func (e *Emitter) EmitError(v error) (err error) {
	return e.EmitString(v.Error())
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
	if n < 0 {
		err = fmt.Errorf("objconv/msgpack: encoding maps of unknown length is not supported (n = %d)", n)
		return
	}

	switch {
	case n <= 15:
		e.b[0] = byte(n) | FixmapTag
		n = 1

	case n <= objutil.Uint16Max:
		e.b[0] = Map16
		putUint16(e.b[1:], uint16(n))
		n = 3

	case n <= objutil.Uint32Max:
		e.b[0] = Map32
		putUint32(e.b[1:], uint32(n))
		n = 5

	default:
		err = fmt.Errorf("objconv/msgpack: map of length %d is too long to be encoded", n)
		return
	}

	_, err = e.w.Write(e.b[:n])
	return
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
	switch {
	case n <= 15:
		e.b[0] = byte(n) | FixarrayTag
		n = 1

	case n <= objutil.Uint16Max:
		e.b[0] = Array16
		putUint16(e.b[1:], uint16(n))
		n = 3

	case n <= objutil.Uint32Max:
		e.b[0] = Array32
		putUint32(e.b[1:], uint32(n))
		n = 5

	default:
		err = fmt.Errorf("objconv/msgpack: array of length %d is too long to be encoded", n)
		return
	}

	_, err = e.w.Write(e.b[:n])
	return
}

var contextPool = sync.Pool{
	New: func() interface{} { return &context{} },
}
