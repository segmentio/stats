package objutil

import (
	"testing"
	"time"
)

var durationTests = []time.Duration{
	0,
	time.Nanosecond,
	time.Microsecond,
	time.Millisecond,
	time.Second,
	time.Minute,
	time.Hour,
	time.Hour + time.Minute + time.Second + (123456789 * time.Nanosecond),
}

func TestAppendDuration(t *testing.T) {
	for _, test := range durationTests {
		t.Run(test.String(), func(t *testing.T) {
			b := AppendDuration(nil, test)
			s := string(b)

			if s != test.String() {
				t.Error(s)
			}
		})
	}
}

func BenchmarkAppendDuration(b *testing.B) {
	for _, test := range durationTests {
		b.Run(test.String(), func(b *testing.B) {
			var a [32]byte
			for i := 0; i != b.N; i++ {
				AppendDuration(a[:0], test)
			}
		})
	}
}
