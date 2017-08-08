package stats_test

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

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
		h1 := &statstest.Handler{}
		h2 := &statstest.Handler{}

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
