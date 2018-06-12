package datadog

import (
	"reflect"
	"testing"
)

func TestParseMetricSuccess(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.s, func(t *testing.T) {
			if m, err := parseMetric(test.s); err != nil {
				t.Error(err)
			} else if !reflect.DeepEqual(m, test.m) {
				t.Errorf("%#v:\n- %#v\n- %#v", test.s, test.m, m)
			}
		})
	}
}

func TestParseMetricFailure(t *testing.T) {
	tests := []string{
		"",
		":10|c",             // missing name
		"name:|c",           // missing value
		"name:abc|c",        // malformed value
		"name:1",            // missing type
		"name:1|",           // missing type
		"name:1|c|???",      // malformed sample rate
		"name:1|c|@abc",     // malformed sample rate
		"name:1|c|@0.5|???", // malformed tags
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			if _, err := parseMetric(test); err == nil {
				t.Errorf("%#v: expected parsing error", test)
			}
		})
	}
}

func BenchmarkParseMetric(b *testing.B) {
	for _, test := range testMetrics {
		b.Run(test.m.Name, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				parseMetric(test.s)
			}
		})
	}
}

func TestParseEventSuccess(t *testing.T) {
	for _, test := range testEvents {
		t.Run(test.s, func(t *testing.T) {
			if e, err := parseEvent(test.s); err != nil {
				t.Error(err)
			} else if !reflect.DeepEqual(e, test.e) {
				t.Errorf("%#v:\n- %#v\n- %#v", test.s, test.e, e)
			}
		})
	}
}

func BenchmarkParseEvent(b *testing.B) {
	for _, test := range testEvents {
		b.Run(test.e.Title, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				parseEvent(test.s)
			}
		})
	}
}
