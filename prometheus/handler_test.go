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
		Buckets: map[stats.Key][]stats.Value{
			stats.Key{Field: "C"}: []stats.Value{
				stats.ValueOf(0.25),
				stats.ValueOf(0.5),
				stats.ValueOf(0.75),
				stats.ValueOf(1.0),
			},
		},
	}

	input := []stats.Measure{
		{Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)}},
		{Fields: []stats.Field{stats.MakeField("A", 2, stats.Counter)}},
		{Fields: []stats.Field{stats.MakeField("C", 0.1, stats.Histogram)}},
		{Fields: []stats.Field{stats.MakeField("B", 1, stats.Gauge)}, Tags: []stats.Tag{stats.T("a", "1"), stats.T("b", "2")}},
		{Fields: []stats.Field{stats.MakeField("A", 4, stats.Counter)}, Tags: []stats.Tag{stats.T("id", "123")}},
		{Fields: []stats.Field{stats.MakeField("B", 42, stats.Gauge)}, Tags: []stats.Tag{stats.T("a", "1")}},
		{Fields: []stats.Field{stats.MakeField("C", 0.1, stats.Histogram)}},
		{Fields: []stats.Field{stats.MakeField("B", 21, stats.Gauge)}, Tags: []stats.Tag{stats.T("a", "1"), stats.T("b", "2")}},
		{Fields: []stats.Field{stats.MakeField("C", 0.5, stats.Histogram)}},
		{Fields: []stats.Field{stats.MakeField("C", 10, stats.Histogram)}},
	}

	handler.HandleMeasures(now, input...)

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
C_bucket{le="0.5"} 3 1496614320000
C_bucket{le="0.75"} 3 1496614320000
C_bucket{le="1"} 3 1496614320000
C_count 4 1496614320000
C_sum 10.7 1496614320000
`

	if s := string(b); s != expects {
		t.Error("bad output:")
		t.Log("expected:", expects)
		t.Log("found:", s)
	}
}

func BenchmarkHandleMetric(b *testing.B) {
	now := time.Now()

	buckets := map[stats.Key][]stats.Value{
		stats.Key{Field: "C"}: []stats.Value{
			stats.ValueOf(0.25),
			stats.ValueOf(0.5),
			stats.ValueOf(0.75),
			stats.ValueOf(1.0),
		},
	}

	metrics := []stats.Measure{
		{
			Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("a", "1"), stats.T("b", "2")},
		},
		{
			Fields: []stats.Field{stats.MakeField("B", 1, stats.Gauge)},
			Tags:   []stats.Tag{stats.T("a", "1"), stats.T("b", "2")},
		},
		{
			Fields: []stats.Field{stats.MakeField("C", 0.1, stats.Histogram)},
			Tags:   []stats.Tag{stats.T("a", "1"), stats.T("b", "2")},
		},
	}

	for _, metric := range metrics {
		b.Run(metric.Fields[0].Type().String(), func(b *testing.B) {
			handler := &Handler{
				Buckets: buckets,
			}

			for i := 0; i != b.N; i++ {
				handler.HandleMeasures(now, metric)
			}
		})
	}
}
