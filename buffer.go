package stats

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
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
	// decision of
	BufferSize int

	// The Serializer used to write the measures.
	//
	// This field cannot be nil.
	Serializer Serializer

	buffer unsafe.Pointer
	join   sync.WaitGroup
}

// HandleMeasures satisfies the Handler interface.
func (h *Buffer) HandleMeasures(time time.Time, measures ...Measure) {
	if len(measures) == 0 {
		return
	}

	size := h.BufferSize
	if size == 0 {
		size = 1024
	}

	c := chunkPool.Get().(*chunk)
	c.b = h.Serializer.AppendMeasures(c.b[:0], time, measures...)
	l := int32(len(c.b))

	// Chunks that are already larger than the maximum buffer size are directly
	// passed to the serializer, no need to aggregate them in a larger buffer.
	if int(l) >= size {
		h.join.Add(1)
		go h.writeChunk(c)
		return
	}

	for {
		b := h.loadBuffer()

		if b == nil {
			b = bufferPool.Get().(*buffer)
			b.init(size)

			if !h.compareAndSwapBuffer(nil, b) {
				bufferPool.Put(b)
				continue
			}
		}

		b.m.RLock()
		m := int32(len(b.b))
		n := atomic.AddInt32(&b.n, l)

		if n >= 0 && n <= m {
			copy(b.b[n-l:n], c.b)
			b.m.RUnlock()
			break
		}

		b.m.RUnlock()

		if n < 0 || (n-l) > m {
			// Only the goroutine that first overflows the buffer size is in
			// charge of writing the buffer, the others go back and try to
			// acquire a new one to copy their measures into.
			continue
		}

		if h.compareAndSwapBuffer(b, nil) {
			b.waitReleaseRLock()
			h.join.Add(1)
			go h.writeBuffer(b, int(n-l))
		}
	}

	chunkPool.Put(c)
}

// Flush satisfies the Flusher interface.
func (h *Buffer) Flush() {
	for {
		b := h.loadBuffer()
		if b == nil {
			break
		}
		if h.compareAndSwapBuffer(b, nil) {
			n := b.waitReleaseRLockAndFinalize()
			if n <= len(b.b) {
				h.join.Add(1)
				go h.writeBuffer(b, n)
			}
		}
	}
	h.join.Wait()
}

func (h *Buffer) loadBuffer() *buffer {
	return (*buffer)(atomic.LoadPointer(&h.buffer))
}

func (h *Buffer) compareAndSwapBuffer(old *buffer, new *buffer) bool {
	return atomic.CompareAndSwapPointer(&h.buffer,
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}

func (h *Buffer) writeBuffer(b *buffer, n int) {
	h.Serializer.Write(b.b[:n])
	h.join.Done()
	bufferPool.Put(b)
}

func (h *Buffer) writeChunk(c *chunk) {
	h.Serializer.Write(c.b)
	h.join.Done()
	chunkPool.Put(c)
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
	m sync.RWMutex
	n int32
}

func (buf *buffer) init(size int) {
	buf.m.Lock()
	if cap(buf.b) < size {
		buf.b = make([]byte, 0, size)
	}
	buf.b = buf.b[:size]
	buf.n = 0
	buf.m.Unlock()
}

func (buf *buffer) waitReleaseRLock() {
	buf.m.Lock()
	buf.m.Unlock()
}

func (buf *buffer) waitReleaseRLockAndFinalize() int {
	buf.m.Lock()
	n := buf.n
	buf.n = int32(len(buf.b))
	buf.m.Unlock()
	return int(n)
}

type chunk struct {
	b []byte
}

var bufferPool = sync.Pool{
	New: func() interface{} { return &buffer{} },
}

var chunkPool = sync.Pool{
	New: func() interface{} { return &chunk{b: make([]byte, 1024)} },
}
