package statstest

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertoforce/stats"
)

var _ stats.Handler = (*Handler)(nil)
var _ stats.Flusher = (*Handler)(nil)

// Handler is a stats handler that can record measures for inspection.
type Handler struct {
	sync.Mutex
	measures []stats.Measure
	flush    int32
}

// HandleMeasures process a variadic list of stats.Measure.
func (h *Handler) HandleMeasures(time time.Time, measures ...stats.Measure) {
	h.Lock()
	for _, m := range measures {
		h.measures = append(h.measures, m.Clone())
	}
	h.Unlock()
}

// Measures returns a copy of the handled measures.
func (h *Handler) Measures() []stats.Measure {
	h.Lock()
	m := make([]stats.Measure, len(h.measures))
	copy(m, h.measures)
	h.Unlock()
	return m
}

// Flush Increments Flush counter.
func (h *Handler) Flush() {
	atomic.AddInt32(&h.flush, 1)
}

// FlushCalls returns the number of times `Flush` has been invoked.
func (h *Handler) FlushCalls() int {
	return int(atomic.LoadInt32(&h.flush))
}

// Clear removes all measures held by Handler.
func (h *Handler) Clear() {
	h.Lock()
	h.measures = h.measures[:0]
	h.Unlock()
}
