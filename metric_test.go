package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestMakeOpts(t *testing.T) {
	tests := []struct {
		name string
		help string
		tags Tags
	}{
		{
			name: "",
			help: "",
			tags: nil,
		},
		{
			name: "hello",
			help: "world",
			tags: Tags{{"hello", "world"}},
		},
	}

	for _, test := range tests {
		opts := MakeOpts(test.name, test.help, test.tags...)

		if opts.Name != test.name {
			t.Errorf("invalid opts name: %#v != %#v", test.name, opts.Name)
		}

		if opts.Help != test.help {
			t.Errorf("invalid opts help: %#v != %#v", test.help, opts.Help)
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
			m: NewGauge(Opts{}, nil),
			s: "gauge",
		},
		{
			m: NewCounter(Opts{}, nil),
			s: "counter",
		},
		{
			m: NewHistogram(Opts{}, nil),
			s: "histogram",
		},
		{
			m: NewTimer(time.Time{}, Opts{}, nil),
			s: "timer",
		},
	}

	for _, test := range tests {
		if s := test.m.Type(); s != test.s {
			t.Errorf("invalid type for %v: %s != %s", test.m, test.s, s)
		}
	}
}
