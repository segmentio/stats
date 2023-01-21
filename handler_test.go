package stats_test

import (
	"testing"
	"time"

	"github.com/vertoforce/stats"
	"github.com/vertoforce/stats/statstest"

	"github.com/stretchr/testify/assert"
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

func TestFilteredHandler(t *testing.T) {
	t.Run("calling HandleMeasures on a filteredHandler processes the measures with the filter", func(t *testing.T) {
		handler := &statstest.Handler{}
		filter := func(ms []stats.Measure) []stats.Measure {
			measures := make([]stats.Measure, 0, len(ms))
			for _, m := range ms {
				fields := make([]stats.Field, 0, len(m.Fields))
				for _, f := range m.Fields {
					if f.Name == "a" {
						fields = append(fields, f)
					}
				}
				if len(fields) > 0 {
					measures = append(measures, stats.Measure{Name: m.Name, Fields: fields, Tags: m.Tags})
				}
			}
			return measures
		}
		fh := stats.FilteredHandler(handler, filter)
		stats.Register(fh)

		stats.Observe("b", 1.23)
		assert.Equal(t, []stats.Measure{}, handler.Measures())

		stats.Observe("a", 1.23)
		assert.Equal(t, []stats.Measure{
			{
				Name:   "stats.test",
				Fields: []stats.Field{stats.MakeField("a", 1.23, stats.Histogram)},
				Tags:   nil,
			},
		}, handler.Measures())

		stats.Incr("b")
		assert.Equal(t, []stats.Measure{
			{
				Name:   "stats.test",
				Fields: []stats.Field{stats.MakeField("a", 1.23, stats.Histogram)},
				Tags:   nil,
			},
		}, handler.Measures())

		stats.Incr("a")
		assert.Equal(t, []stats.Measure{
			{
				Name:   "stats.test",
				Fields: []stats.Field{stats.MakeField("a", 1.23, stats.Histogram)},
				Tags:   nil,
			},
			{
				Name:   "stats.test",
				Fields: []stats.Field{stats.MakeField("a", 1, stats.Counter)},
				Tags:   nil,
			},
		}, handler.Measures())
	})

	t.Run("calling Flush on a FilteredHandler flushes the underlying handler", func(t *testing.T) {
		h := &statstest.Handler{}

		m := stats.FilteredHandler(h, func(ms []stats.Measure) []stats.Measure { return ms })
		flush(m)

		assert.EqualValues(t, 1, h.FlushCalls(), "Flush should be called once")
	})
}
