package prometheus

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestTextWriter(t *testing.T) {
	tests := []struct {
		metric Metric
		string string
	}{
		{
			metric: Metric{
				Name: "hello_world",
			},
			string: `hello_world 0
`,
		},

		{
			metric: Metric{
				Name: "hello_world",
				Help: "this is a metric",
			},
			string: `# HELP hello_world this is a metric
hello_world 0
`,
		},

		{
			metric: Metric{
				Name: "hello_world",
				Type: "gauge",
			},
			string: `# TYPE hello_world gauge
hello_world 0
`,
		},

		{
			metric: Metric{
				Name:  "hello_world",
				Value: 1.234,
			},
			string: `hello_world 1.234
`,
		},

		{
			metric: Metric{
				Name:   "hello_world",
				Labels: Labels{{"answer", "42"}},
			},
			string: `hello_world{answer="42"} 0
`,
		},

		{
			metric: Metric{
				Name: "hello_world",
				Time: time.Unix(1, 0),
			},
			string: `hello_world 0 1000
`,
		},

		{
			metric: Metric{
				Name:   "hello_world",
				Type:   "gauge",
				Help:   "this is a metric",
				Value:  1.234,
				Time:   time.Unix(1, 0),
				Labels: Labels{{"answer", "42"}},
			},
			string: `# HELP hello_world this is a metric
# TYPE hello_world gauge
hello_world{answer="42"} 1.234 1000
`,
		},
	}

	for _, test := range tests {
		b := &bytes.Buffer{}
		w := NewTextWriter(b)
		w.WriteMetric(test.metric)

		if s := b.String(); s != test.string {
			t.Errorf("the text writer produced an invalid representation of a metric:\n- expected: %s\n- found: %s", test.string, s)
		}
	}
}

func TestMakeMetric(t *testing.T) {
	now := time.Now()

	tests := []struct {
		m stats.Metric
		v float64
		t time.Time
		x Metric
	}{
		{
			m: stats.NewGauge(stats.MakeOpts("test", stats.Tag{"hello", "world"})),
			v: 1,
			t: now,
			x: Metric{
				Name:   "test",
				Type:   "gauge",
				Value:  1,
				Time:   now,
				Labels: Labels{{"hello", "world"}},
				key:    `test{hello="world"}`,
				sum:    1,
				count:  1,
			},
		},
	}

	for _, test := range tests {
		if x := makeMetric(test.m, test.v, test.t); !reflect.DeepEqual(x, test.x) {
			t.Errorf("invalid metric: %#v != %#v", test.x, x)
		}
	}
}

func TestMetricApply(t *testing.T) {
	tests := []struct {
		m Metric
		v float64
		x float64
	}{
		{
			m: Metric{Type: "gauge", Value: 1},
			v: 2,
			x: 2,
		},
		{
			m: Metric{Type: "counter", Value: 1},
			v: 2,
			x: 3,
		},
	}

	for _, test := range tests {
		if test.m.apply(test.v); test.m.Value != test.x {
			t.Errorf("invalid metric value after applying %s: %v != %v", test.m.Type, test.x, test.m.Value)
		}
	}
}

func TestMetricStoreInsertSuccess(t *testing.T) {
	store := newMetricStore()
	tests := []Metric{
		{
			Name:  "metric_1",
			Type:  "gauge",
			Value: 1,
			key:   "metric_1",
			sum:   1,
			count: 1,
		},
		{
			Name:   "metric_1",
			Type:   "gauge",
			Value:  1,
			Labels: Labels{{"hello", "world"}},
			key:    `metric_1{hello="world"}`,
			sum:    1,
			count:  1,
		},
		{
			Name:  "metric_2",
			Type:  "gauge",
			Value: 42,
			key:   "metric_2",
			sum:   42,
			count: 1,
		},
		{
			Name:  "metric_1",
			Type:  "gauge",
			Value: 10,
			key:   "metric_1",
			sum:   11,
			count: 2,
		},
	}

	for _, m := range tests {
		if err := store.insert(m); err != nil {
			t.Error(err)
		}
	}

	if metrics := store.snapshot(); !reflect.DeepEqual(metrics, []Metric{
		tests[3],
		tests[1],
		tests[2],
	}) {
		t.Errorf("invalid metric snapshot: %v", metrics)
	}
}

func TestMetricStoreInsertFailure(t *testing.T) {
	store := newMetricStore()

	m1 := Metric{
		Name:  "metric_1",
		Type:  "gauge",
		Value: 1,
		key:   "metric_1",
		sum:   1,
		count: 1,
	}

	m2 := Metric{
		Name:  "metric_1",
		Type:  "counter",
		Value: 1,
		key:   "metric_1",
		sum:   1,
		count: 1,
	}

	if err := store.insert(m1); err != nil {
		t.Error(err)
	}

	if err := store.insert(m2); err == nil {
		t.Error("expected error when inserting metrics with the same name but incompatible types")
	}
}

func TestMetricStoreExpire(t *testing.T) {
	now := time.Now()

	store := newMetricStore()
	tests := []Metric{
		{
			Name:  "metric_1",
			Type:  "gauge",
			Value: 1,
			Time:  now.Add(-time.Second),
			key:   "metric_1",
			sum:   1,
			count: 1,
		},
		{
			Name:   "metric_1",
			Type:   "gauge",
			Value:  1,
			Time:   now.Add(-time.Second),
			Labels: Labels{{"hello", "world"}},
			key:    `metric_1{hello="world"}`,
			sum:    1,
			count:  1,
		},
		{
			Name:  "metric_2",
			Type:  "gauge",
			Value: 42,
			Time:  now,
			key:   "metric_2",
			sum:   42,
			count: 1,
		},
		{
			Name:  "metric_1",
			Type:  "gauge",
			Value: 10,
			Time:  now,
			key:   "metric_1",
			sum:   11,
			count: 2,
		},
	}

	for _, m := range tests {
		if err := store.insert(m); err != nil {
			t.Error(err)
		}
	}

	store.expire(now)

	if metrics := store.snapshot(); !reflect.DeepEqual(metrics, []Metric{
		tests[3],
		tests[2],
	}) {
		t.Errorf("invalid metric snapshot: %v", metrics)
	}
}
