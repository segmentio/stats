package prometheus

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestAcceptEncoding(t *testing.T) {
	tests := []struct {
		accept string
		check  string
		expect bool
	}{
		{
			accept: "",
			check:  "gzip",
			expect: false,
		},

		{
			accept: "gzip",
			check:  "gzip",
			expect: true,
		},

		{
			accept: "gzip, deflate, sdch, br",
			check:  "gzip",
			expect: true,
		},

		{
			accept: "deflate, sdch, br",
			check:  "gzip",
			expect: false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s?", test.accept, test.check), func(t *testing.T) {
			if ok := acceptEncoding(test.accept, test.check); ok != test.expect {
				t.Error(ok)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{
		HistogramBuckets: map[string][]float64{"C": {0.25, 0.5, 0.75, 1.0}},
	}

	input := []stats.Metric{
		{Type: stats.CounterType, Name: "A", Value: 1, Time: now},
		{Type: stats.CounterType, Name: "A", Value: 2, Time: now},
		{Type: stats.HistogramType, Name: "C", Value: 0.1, Time: now},
		{Type: stats.GaugeType, Name: "B", Value: 1, Time: now, Tags: []stats.Tag{{"a", "1"}, {"b", "2"}}},
		{Type: stats.CounterType, Name: "A", Value: 4, Time: now, Tags: []stats.Tag{{"id", "123"}}},
		{Type: stats.GaugeType, Name: "B", Value: 42, Time: now, Tags: []stats.Tag{{"a", "1"}}},
		{Type: stats.HistogramType, Name: "C", Value: 0.1, Time: now},
		{Type: stats.GaugeType, Name: "B", Value: 21, Time: now, Tags: []stats.Tag{{"a", "1"}, {"b", "2"}}},
		{Type: stats.HistogramType, Name: "C", Value: 0.5, Time: now},
		{Type: stats.HistogramType, Name: "C", Value: 10, Time: now},
	}

	for i := range input {
		handler.HandleMetric(&input[i])
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	server := httptest.NewServer(mux)
	defer server.Close()

	res, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)

	const expects = `# TYPE A counter
A 3 1496614320000
A{id="123"} 4 1496614320000

# TYPE B gauge
B{a="1"} 42 1496614320000
B{a="1",b="2"} 21 1496614320000

# TYPE C histogram
C_bucket{le="0.25"} 2 1496614320000
C_bucket{le="0.5"} 1 1496614320000
C_bucket{le="0.75"} 0 1496614320000
C_bucket{le="1"} 0 1496614320000
C_count 4 1496614320000
C_sum 10.7 1496614320000
`

	if s := string(b); s != expects {
		t.Error("bad output:")
		t.Log("expected:", expects)
		t.Log("found:", s)
	}
}
