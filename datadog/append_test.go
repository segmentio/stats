package datadog

import "testing"

func TestAppendMetric(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.m.Name, func(b *testing.T) {
			if s := string(appendMetric(nil, test.m)); s != test.s {
				t.Errorf("\n<<< %#v\n>>> %#v", test.s, s)
			}
		})
	}
}

func BenchmarkAppendMetric(b *testing.B) {
	buffer := make([]byte, 4096)

	for _, test := range testMetrics {
		b.Run(test.m.Name, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				appendMetric(buffer[:0], test.m)
			}
		})
	}
}
