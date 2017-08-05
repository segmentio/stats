package redisstats

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

type testMeasureHandler struct {
	sync.Mutex
	measures []stats.Measure
}

func (h *testMeasureHandler) Measures() []stats.Measure {
	h.Lock()
	c := make([]stats.Measure, len(h.measures))
	copy(c, h.measures)
	h.Unlock()
	return c
}

func (h *testMeasureHandler) HandleMeasures(time time.Time, measures ...stats.Measure) {
	h.Lock()
	for _, m := range measures {
		h.measures = append(h.measures, m.Clone())
	}
	h.Unlock()
}

func validateMeasure(t *testing.T, found stats.Measure, expect stats.Measure) {
	if !reflect.DeepEqual(found, expect) {
	}

}
