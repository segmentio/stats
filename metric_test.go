package stats

import "testing"

func TestMetricTypeString(t *testing.T) {
	tests := []struct {
		t MetricType
		s string
	}{
		{CounterType, "counter"},
		{GaugeType, "gauge"},
		{HistogramType, "histogram"},
		{MetricType(-1), "unknown"},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			if s := test.t.String(); s != test.s {
				t.Error(s)
			}
		})
	}
}
