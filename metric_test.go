package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestMakeOpts(t *testing.T) {
	tests := []struct {
		name string
		tags Tags
	}{
		{
			name: "",
			tags: nil,
		},
		{
			name: "hello",
			tags: Tags{{"hello", "world"}},
		},
	}

	for _, test := range tests {
		opts := MakeOpts(test.name, test.tags...)

		if opts.Name != test.name {
			t.Errorf("invalid opts name: %#v != %#v", test.name, opts.Name)
		}

		if !reflect.DeepEqual(opts.Tags, test.tags) {
			t.Errorf("invalid opts tags: %#v != %#v", test.tags, opts.Tags)
		}
	}
}

func TestMetricType(t *testing.T) {
	tests := []struct {
		m Metric
		s string
	}{
		{
			m: NewGauge(nil, Opts{}),
			s: "gauge",
		},
		{
			m: NewCounter(nil, Opts{}),
			s: "counter",
		},
		{
			m: NewHistogram(nil, Opts{}),
			s: "histogram",
		},
		{
			m: NewTimer(time.Time{}, nil, Opts{}),
			s: "timer",
		},
	}

	for _, test := range tests {
		if s := test.m.Type(); s != test.s {
			t.Errorf("invalid type for %v: %s != %s", test.m, test.s, s)
		}
	}
}
