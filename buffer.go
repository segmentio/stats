package stats

import (
	"io"
	"sync"
	"time"
)

// Buffer is the implementation of a measure handler which uses a Serializer to
// serialize the metric into a memory buffer and write them once the buffer has
// reached a target size.
type Buffer struct {
	// Target size of the memory buffer where metrics are serialized.
	//
	// If left to zero, a size of 1024 bytes is used as default (this is low,
	// you should set this value).
	//
	// Note that if the buffer size is small, the program may generate metrics
	// that don't fit into the configured buffer size. In that case the buffer
	// will still pass the serialized byte slice to its Serializer to leave the
	// decision of accepting or rejecting the metrics.
	BufferSize int

	// The Serializer used to write the measures.
	//
	// This field cannot be nil.
	Serializer Serializer

	mutex  sync.Mutex
	buffer *buffer
}

// HandleMeasures satisfies the Handler interface.
func (b *Buffer) HandleMeasures(time time.Time, measures ...Measure) {
	if len(measures) == 0 {
		return
	}

	size := b.BufferSize
	if size == 0 {
		size = 1024
	}

	chunk := acquireChunk()
	chunk.reset()
	chunk.append(b.Serializer, time, measures...)

	// Chunks that are already larger than the maximum buffer size are directly
	// passed to the serializer, no need to aggregate them in a larger buffer.
	if chunk.len() >= size {
		chunk.writeTo(b.Serializer)
	} else {
		for {
			var flush *buffer

			b.mutex.Lock()

			if b.buffer == nil {
				b.buffer = acquireBuffer()
				b.buffer.reset(size)
			}

			if (b.buffer.len() + chunk.len()) > b.buffer.cap() {
				b.buffer, flush = nil, b.buffer
			} else {
				b.buffer.append(chunk)
			}

			b.mutex.Unlock()

			if flush == nil {
				break
			}

			flush.writeTo(b.Serializer)
			releaseBuffer(flush)
		}
	}

	releaseChunk(chunk)
}

// Flush satisfies the Flusher interface.
func (b *Buffer) Flush() {
	b.mutex.Lock()

	if buffer := b.buffer; buffer != nil {
		b.buffer = nil
		buffer.writeTo(b.Serializer)
		releaseBuffer(buffer)
	}

	b.mutex.Unlock()
}

// The Serializer interface is used to abstract the logic of serializing
// measures.
type Serializer interface {
	io.Writer

	// Appends the serialized representation of the given measures into b.
	//
	// The method must not retain any of the arguments.
	AppendMeasures(b []byte, time time.Time, measures ...Measure) []byte
}

type buffer struct {
	b []byte
}

func (buf *buffer) reset(size int) {
	if cap(buf.b) < size {
		buf.b = make([]byte, 0, size)
	} else {
		buf.b = buf.b[:0]
	}
}

func (buf *buffer) append(c *chunk) {
	buf.b = append(buf.b, c.b...)
}

func (buf *buffer) len() int {
	return len(buf.b)
}

func (buf *buffer) cap() int {
	return cap(buf.b)
}

func (buf *buffer) writeTo(w io.Writer) {
	w.Write(buf.b)
}

type chunk struct {
	b []byte
}

func (c *chunk) reset() {
	c.b = c.b[:0]
}

func (c *chunk) append(s Serializer, t time.Time, m ...Measure) {
	c.b = s.AppendMeasures(c.b, t, m...)
}

func (c *chunk) len() int {
	return len(c.b)
}

func (c *chunk) writeTo(w io.Writer) {
	w.Write(c.b)
}

var bufferPool = sync.Pool{
	New: func() interface{} { return &buffer{} },
}

var chunkPool = sync.Pool{
	New: func() interface{} { return &chunk{b: make([]byte, 1024)} },
}

func acquireBuffer() *buffer  { return bufferPool.Get().(*buffer) }
func releaseBuffer(b *buffer) { bufferPool.Put(b) }

func acquireChunk() *chunk  { return chunkPool.Get().(*chunk) }
func releaseChunk(c *chunk) { chunkPool.Put(c) }
