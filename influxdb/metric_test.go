package influxdb

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

var (
	timestamp   = time.Date(2017, 7, 23, 3, 36, 0, 123456789, time.UTC)
	testMetrics = []struct {
		m stats.Metric
		s string
	}{
		{
			m: stats.Metric{
				Name:  "request.count",
				Time:  timestamp,
				Value: 0.5,
			},
			s: `request.count value=0.5 1500780960123456789`,
		},
		{
			m: stats.Metric{
				Namespace: "testing",
				Name:      "request.count",
				Time:      timestamp,
				Value:     0.5,
			},
			s: `testing.request.count value=0.5 1500780960123456789`,
		},
		{
			m: stats.Metric{
				Name:  "request.count",
				Time:  timestamp,
				Value: 0.5,
				Tags: []stats.Tag{
					{"hello", "world"},
					{"answer", "42"},
				},
			},
			s: `request.count,hello=world,answer=42 value=0.5 1500780960123456789`,
		},
	}
)

func TestAppendMetric(t *testing.T) {
	for _, test := range testMetrics {
		t.Run(test.s, func(t *testing.T) {
			if s := string(appendMetric(nil, &test.m)); s != (test.s + "\n") {
				t.Error("bad metric representation:")
				t.Log("expected:", test.s)
				t.Log("found:   ", s)
			}
		})
	}
}

func TestMetricsReaderRead(t *testing.T) {
	m := newMetrics(100)
	s := ""

	for _, test := range testMetrics {
		m.append(&test.m)
		s += test.s
		s += "\n"
	}

	b := &bytes.Buffer{}
	w := newMetricsReader(m)

	for {
		var a [128]byte
		n, err := w.Read(a[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Error(err)
		}
		b.Write(a[:n])
	}

	if b.String() != s {
		t.Error("bad metrics read:")
		t.Log("expected:", s)
		t.Log("found:   ", b.String())
	}
}

func TestMetricsReaderWriteTo(t *testing.T) {
	m := newMetrics(100)
	s := ""

	for _, test := range testMetrics {
		m.append(&test.m)
		s += test.s
		s += "\n"
	}

	b := &bytes.Buffer{}
	w := newMetricsReader(m)
	w.WriteTo(b)

	if b.String() != s {
		t.Error("bad metrics read:")
		t.Log("expected:", s)
		t.Log("found:   ", b.String())
	}
}

func BenchmarkAppendMetrics(b *testing.B) {
	a := make([]byte, 1024)

	for _, test := range testMetrics {
		b.Run(test.s, func(b *testing.B) {
			for i := 0; i != b.N; i++ {
				appendMetric(a[:0], &test.m)
			}
		})
	}
}
