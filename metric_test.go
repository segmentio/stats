package stats

import (
	"reflect"
	"testing"
	"time"
)

func TestNormalizeSampleRate(t *testing.T) {
	tests := []struct {
		rate float64
		norm float64
	}{
		{
			rate: 0,
			norm: 1,
		},
		{
			rate: 1,
			norm: 1,
		},
		{
			rate: -1,
			norm: 1,
		},
		{
			rate: 2,
			norm: 1,
		},
		{
			rate: 0.5,
			norm: 0.5,
		},
	}

	for _, test := range tests {
		if n := normalizeSampleRate(test.rate); n != test.norm {
			t.Errorf("%v: invalid normalized sample rate: %v != %v", test.rate, test.norm, n)
		}
	}
}

func TestPassSampleRate(t *testing.T) {
	tests := []struct {
		rate float64
		rand func() float64
		pass bool
	}{
		{
			rate: 1,
			rand: func() float64 { return 0 },
			pass: true,
		},
		{
			rate: 0.1,
			rand: func() float64 { return 0.1 },
			pass: false,
		},
		{
			rate: 0.1,
			rand: func() float64 { return 0.05 },
			pass: true,
		},
	}

	for _, test := range tests {
		if pass := passSampleRate(test.rate, test.rand); pass != test.pass {
			t.Errorf("%v: sample rate check failed: expected %v but got %v", test.rate, test.pass, pass)
		}
	}
}

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

func TestMetricSample(t *testing.T) {
	tests := []struct {
		m Metric
		s float64
	}{
		{
			m: NewGauge(Opts{}, nil),
			s: 1,
		},
		{
			m: NewCounter(Opts{Sample: 0.1}, nil),
			s: 0.1,
		},
	}

	for _, test := range tests {
		if s := test.m.Sample(); s != test.s {
			t.Errorf("invalid sample for %v: %s != %s", test.m, test.s, s)
		}
	}
}
