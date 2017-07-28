package stats_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

type testHandler struct {
	mutex    sync.Mutex
	measures []stats.Measure
	flush    int32
}

func (h *testHandler) HandleMeasures(time time.Time, measures ...stats.Measure) {
	h.mutex.Lock()
	for _, m := range measures {
		h.measures = append(h.measures, m.Clone())
	}
	h.mutex.Unlock()
}

func (h *testHandler) Measures() []stats.Measure {
	h.mutex.Lock()
	measures := make([]stats.Measure, len(h.measures))
	copy(measures, h.measures)
	h.mutex.Unlock()
	return measures
}

func (h *testHandler) Flush() {
	atomic.AddInt32(&h.flush, 1)
}

func (h *testHandler) FlushCalls() int {
	return int(atomic.LoadInt32(&h.flush))
}

func TestMultiHandler(t *testing.T) {
	t.Run("calling HandleMeasures on a multi-handler dispatches to each handler", func(t *testing.T) {
		n := 0
		f := stats.HandlerFunc(func(time time.Time, measures ...stats.Measure) { n++ })
		m := stats.MultiHandler(f, f, f)

		m.HandleMeasures(time.Now())

		if n != 3 {
			t.Error("bad number of calls to HandleMeasures:", n)
		}
	})

	t.Run("calling Flush on a multi-handler flushes each handler", func(t *testing.T) {
		h1 := &testHandler{}
		h2 := &testHandler{}

		m := stats.MultiHandler(h1, h2)
		flush(m)
		flush(m)

		n1 := h1.FlushCalls()
		n2 := h2.FlushCalls()

		if n1 != 2 {
			t.Error("bad number of calls to Flush:", n1)
		}

		if n2 != 2 {
			t.Error("bad number of calls to Flush:", n2)
		}
	})
}

func flush(h stats.Handler) {
	if f, ok := h.(stats.Flusher); ok {
		f.Flush()
	}
}
