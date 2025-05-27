package cbor

import (
	"io"
	"math"
	"time"
	"unsafe"

	"github.com/segmentio/stats/v5/util/objconv/objutil"
)

// Emitter implements a MessagePack emitter that satisfies the objconv.Emitter
// interface.
type Emitter struct {
	w io.Writer
	b [16]byte

	// This stack is used to keep track of the array map lengths being parsed.
	// The sback array is the initial backend array for the stack.
	stack []int
	sback [16]int
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
	e.b[0] = majorByte(majorType7, svNull)
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitBool(v bool) (err error) {
	if v {
		e.b[0] = majorByte(majorType7, svTrue)
	} else {
		e.b[0] = majorByte(majorType7, svFalse)
	}
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitInt(v int64, _ int) (err error) {
	if v >= 0 {
		return e.emitUint(majorType0, uint64(v))
	}
	return e.emitUint(majorType1, uint64(-(v + 1)))
}

func (e *Emitter) EmitUint(v uint64, _ int) (err error) {
	return e.emitUint(majorType0, v)
}

func (e *Emitter) EmitFloat(v float64, bitSize int) (err error) {
	n := 0

	if bitSize == 32 {
		n = 5
		e.b[0] = majorByte(majorType7, svFloat32)
		putUint32(e.b[1:], math.Float32bits(float32(v)))
	} else {
		n = 9
		e.b[0] = majorByte(majorType7, svFloat64)
		putUint64(e.b[1:], math.Float64bits(v))
	}

	_, err = e.w.Write(e.b[:n])
	return
}

func (e *Emitter) EmitString(v string) (err error) {
	if err = e.emitUint(majorType3, uint64(len(v))); err != nil {
		return
	}
	b := unsafe.Slice(unsafe.StringData(v), len(v))
	_, err = e.w.Write(b)
	return
}

func (e *Emitter) EmitBytes(v []byte) (err error) {
	if err = e.emitUint(majorType2, uint64(len(v))); err != nil {
		return
	}
	if len(v) != 0 {
		_, err = e.w.Write(v)
	}
	return
}

func (e *Emitter) EmitTime(v time.Time) (err error) {
	e.b[0] = majorByte(majorType6, tagDateTime)

	if _, err = e.w.Write(e.b[:1]); err != nil {
		return
	}

	var a [64]byte
	var b = v.AppendFormat(a[:0], time.RFC3339Nano)

	if err = e.emitUint(majorType3, uint64(len(b))); err != nil {
		return
	}

	_, err = e.w.Write(append(e.b[:0], b...))
	return
}

func (e *Emitter) EmitDuration(v time.Duration) (err error) {
	return e.EmitString(string(objutil.AppendDuration(e.b[:0], v)))
}

func (e *Emitter) EmitError(v error) (err error) {
	return e.EmitString(v.Error())
}

func (e *Emitter) EmitArrayBegin(n int) (err error) {
	e.stack = append(e.stack, n)

	if n >= 0 {
		return e.emitUint(majorType4, uint64(n))
	}

	e.b[0] = majorByte(majorType4, 31)
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitArrayEnd() (err error) {
	i := len(e.stack) - 1
	n := e.stack[i]
	e.stack = e.stack[:i]

	if n < 0 {
		e.b[0] = 0xFF
		_, err = e.w.Write(e.b[:1])
	}
	return
}

func (e *Emitter) EmitArrayNext() (err error) {
	return
}

func (e *Emitter) EmitMapBegin(n int) (err error) {
	e.stack = append(e.stack, n)

	if n >= 0 {
		return e.emitUint(majorType5, uint64(n))
	}

	e.b[0] = majorByte(majorType5, 31)
	_, err = e.w.Write(e.b[:1])
	return
}

func (e *Emitter) EmitMapEnd() (err error) {
	i := len(e.stack) - 1
	n := e.stack[i]
	e.stack = e.stack[:i]

	if n < 0 {
		e.b[0] = 0xFF
		_, err = e.w.Write(e.b[:1])
	}
	return
}

func (e *Emitter) EmitMapValue() (err error) {
	return
}

func (e *Emitter) EmitMapNext() (err error) {
	return
}

func (e *Emitter) emitUint(m byte, v uint64) (err error) {
	var n int

	switch {
	case v <= 23:
		n = 1
		e.b[0] = majorByte(m, byte(v))

	case v <= objutil.Uint8Max:
		n = 2
		e.b[0] = majorByte(m, iUint8)
		e.b[1] = uint8(v)

	case v <= objutil.Uint16Max:
		n = 3
		e.b[0] = majorByte(m, iUint16)
		putUint16(e.b[1:], uint16(v))

	case v <= objutil.Uint32Max:
		n = 5
		e.b[0] = majorByte(m, iUint32)
		putUint32(e.b[1:], uint32(v))

	default:
		n = 9
		e.b[0] = majorByte(m, iUint64)
		putUint64(e.b[1:], v)
	}

	_, err = e.w.Write(e.b[:n])
	return
}
