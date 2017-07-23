package influxdb

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestClient(t *testing.T) {
	transport := &errorCaptureTransport{
		RoundTripper: http.DefaultTransport,
	}

	client := NewClientWith(ClientConfig{
		Address:       DefaultAddress,
		Database:      "test-db",
		BatchSize:     100,
		FlushInterval: 100 * time.Millisecond,
		Transport:     transport,
	})

	if err := client.CreateDB("test-db"); err != nil {
		t.Error(err)
	}

	for i := 0; i != 1000; i++ {
		client.HandleMetric(&stats.Metric{
			Name:  "test.metric",
			Value: float64(i),
			Time:  time.Now(),
		})
	}

	client.Close()

	if transport.err != nil {
		t.Error(transport.err)
	}
}

func BenchmarkClient(b *testing.B) {
	c := NewClientWith(ClientConfig{
		Address:   DefaultAddress,
		BatchSize: 1000,
		Transport: &discardTransport{},
	})

	m := &stats.Metric{
		Namespace: "benchmark",
		Name:      "test.metric",
		Value:     42,
		Time:      time.Now(),
		Tags: []stats.Tag{
			{"hello", "world"},
			{"answer", "42"},
		},
	}

	for i := 0; i != b.N; i++ {
		c.HandleMetric(m)
	}
}

type errorCaptureTransport struct {
	http.RoundTripper
	err error
}

func (t *errorCaptureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := t.RoundTripper.RoundTrip(req)

	if t.err == nil {
		if err != nil {
			t.err = err
		} else if res.StatusCode >= 300 {
			t.err = fmt.Errorf("%s %s: %d", req.Method, req.URL, res.StatusCode)
		}
	}

	return res, err
}

type discardTransport struct {
}

func (t *discardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}
