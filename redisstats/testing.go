package redisstats

import (
	"sync"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

type testMetricHandler struct {
	sync.Mutex
	metrics []stats.Metric
}

func (h *testMetricHandler) HandleMetric(m *stats.Metric) {
	h.Lock()
	c := *m
	c.Tags = append([]stats.Tag{}, m.Tags...)
	c.Time = time.Time{} // discard because it's unpredicatable
	h.metrics = append(h.metrics, c)
	h.Unlock()
}

func validateMetric(t *testing.T, metric stats.Metric, name string, tags []stats.Tag, metricType stats.MetricType) {
	if name != metric.Name {
		t.Errorf("Expected metric name '%s', found '%s'", name, metric.Name)
	}
	for _, desiredTag := range tags {
		foundTag := false
		for _, actualTag := range metric.Tags {
			if actualTag.Name == desiredTag.Name && actualTag.Value == desiredTag.Value {
				foundTag = true
			}
		}
		if !foundTag {
			t.Errorf("Metric did not contain tag '%s' with value '%s'", desiredTag.Name, desiredTag.Value)
		}
	}
	if metricType != metric.Type {
		t.Errorf("Incorrect metric type %d", metric.Type)
	}
}
