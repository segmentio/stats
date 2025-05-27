package resp

import (
	"bytes"
	"io"
	"sync"

	"github.com/segmentio/stats/v5/util/objconv"
)

// NewEncoder returns a new RESP encoder that writes to w.
func NewEncoder(w io.Writer) *objconv.Encoder {
	return objconv.NewEncoder(NewEmitter(w))
}

// NewStreamEncoder returns a new RESP stream encoder that writes to w.
func NewStreamEncoder(w io.Writer) *objconv.StreamEncoder {
	return objconv.NewStreamEncoder(NewEmitter(w))
}

// Marshal writes the RESP representation of v to a byte slice returned in b.
func Marshal(v interface{}) (b []byte, err error) {
	m := marshalerPool.Get().(*marshaler)
	m.b.Truncate(0)

	if err = (objconv.Encoder{Emitter: m}).Encode(v); err == nil {
		b = make([]byte, m.b.Len())
		copy(b, m.b.Bytes())
	}

	marshalerPool.Put(m)
	return
}

var marshalerPool = sync.Pool{
	New: func() interface{} { return newMarshaler() },
}

type marshaler struct {
	Emitter
	b bytes.Buffer
}

func newMarshaler() *marshaler {
	m := &marshaler{}
	m.s = m.a[:0]
	m.w = &m.b
	return m
}
