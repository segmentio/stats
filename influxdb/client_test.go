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
		Address:   DefaultAddress,
		Database:  "test-db",
		Transport: transport,
	})

	if err := client.CreateDB("test-db"); err != nil {
		t.Error(err)
	}

	for i := 0; i != 1000; i++ {
		client.HandleMeasures(time.Now(), stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				{Name: "count", Value: stats.ValueOf(5)},
				{Name: "rtt", Value: stats.ValueOf(100 * time.Millisecond)},
			},
			Tags: []stats.Tag{
				stats.T("answer", "42"),
				stats.T("hello", "world"),
			},
		})
	}

	client.Close()

	if transport.err != nil {
		t.Error(transport.err)
	}
}

func BenchmarkClient(b *testing.B) {
	for _, N := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("write a batch of %d measures to a client", N), func(b *testing.B) {
			client := NewClientWith(ClientConfig{
				Address:   DefaultAddress,
				Transport: &discardTransport{},
			})

			timestamp := time.Now()
			measures := make([]stats.Measure, N)

			for i := range measures {
				measures[i] = stats.Measure{
					Name: "benchmark.test.metric",
					Fields: []stats.Field{
						{Name: "value", Value: stats.ValueOf(42)},
					},
					Tags: []stats.Tag{
						stats.T("answer", "42"),
						stats.T("hello", "world"),
					},
				}
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					client.HandleMeasures(timestamp, measures...)
				}
			})
		})
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

type discardTransport struct{}

func (t *discardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}
