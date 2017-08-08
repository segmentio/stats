package statstest

import (
	"sync"
	"time"

	"github.com/segmentio/stats"
)

var _ stats.Handler = (*Handler)(nil)

type Handler struct {
	sync.Mutex
	measures []stats.Measure
}

func (h *Handler) HandleMeasures(time time.Time, measures ...stats.Measure) {
	h.Lock()
	for _, m := range measures {
		h.measures = append(h.measures, m.Clone())
	}
	h.Unlock()
}

func (h *Handler) Measures() []stats.Measure {
	h.Lock()
	m := make([]stats.Measure, len(h.measures))
	copy(m, h.measures)
	h.Unlock()
	return m
}
