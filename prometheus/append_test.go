package prometheus

import (
	"testing"
	"time"
)

var testMetrics = []struct {
	scenario string
	metric   metric
	string   string
}{
	{
		scenario: "simple counter metric",
		metric: metric{
			mtype: counter,
			name:  "hello_world",
			value: 42,
		},
		string: `# TYPE hello_world counter
hello_world 42
`,
	},

	{
		scenario: "simple gauge metric",
		metric: metric{
			mtype: gauge,
			name:  "hello_world",
			value: 42,
		},
		string: `# TYPE hello_world gauge
hello_world 42
`,
	},

	{
		scenario: "simple histogram metric",
		metric: metric{
			mtype:  histogram,
			name:   "hello_world_bucket",
			value:  42,
			labels: labels{{"le", "0.5"}},
		},
		string: `# TYPE hello_world histogram
hello_world_bucket{le="0.5"} 42
`,
	},

	{
		scenario: "counter metric with help, floating point value, labels, and timestamp",
		metric: metric{
			mtype:  counter,
			scope:  "global",
			name:   "hello_world",
			help:   "This is a great metric!\n",
			value:  0.5,
			labels: []label{{"question", "\"???\"\n"}, {"answer", "42"}},
			time:   time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC),
		},
		string: `# HELP global_hello_world This is a great metric!\n
# TYPE global_hello_world counter
global_hello_world{question="\"???\"\n",answer="42"} 0.5 1496614320000
`,
	},
}

func TestAppendMetric(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.scenario, func(t *testing.T) {
			b := appendMetric(nil, test.metric)
			s := string(b)

			if s != test.string {
				t.Error("bad metric representation:")
				t.Log(test.string)
				t.Log(s)
			}
		})
	}
}

func BenchmarkAppendMetric(b *testing.B) {
	a := make([]byte, 8192)

	for _, test := range testMetrics {
		b.Run(test.scenario, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				appendMetric(a[:0], test.metric)
			}
		})
	}
}
