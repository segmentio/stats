package prometheus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
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
			{Field: "C"}: {
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

	b, _ := io.ReadAll(res.Body)

	const expects = `# TYPE A counter
A 3
A{id="123"} 4

# TYPE B gauge
B{a="1"} 42
B{a="1",b="2"} 21

# TYPE C histogram
C_bucket{le="0.25"} 2
C_bucket{le="0.5"} 3
C_bucket{le="0.75"} 3
C_bucket{le="1"} 3
C_bucket{le="+Inf"} 4
C_count 4
C_sum 10.7
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
		{Field: "C"}: {
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

			for b.Loop() {
				handler.HandleMeasures(now, metric)
			}
		})
	}
}

// TestHistogramWithoutRegisteredBuckets covers the fail-silent case: a
// histogram with no entry in the bucket registry used to publish _sum and
// _count with no _bucket series at all, so nothing looked wrong and no
// percentile could be computed.
func TestHistogramWithoutRegisteredBuckets(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{} // no Buckets registry at all

	handler.HandleMeasures(now,
		stats.Measure{Fields: []stats.Field{stats.MakeField("D", 0.003, stats.Histogram)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("D", 0.4, stats.Histogram)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("D", 900, stats.Histogram)}},
	)

	var buf strings.Builder
	handler.WriteStats(&buf)
	out := buf.String()

	for _, want := range []string{
		`D_bucket{le="0.005"} 1`,
		`D_bucket{le="0.5"} 2`,
		`D_bucket{le="+Inf"} 3`,
		`D_count 3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}

	// Every registered boundary plus +Inf must be present.
	if n := strings.Count(out, "D_bucket{"); n != len(DefaultBuckets)+1 {
		t.Errorf("found %d bucket series, expected %d", n, len(DefaultBuckets)+1)
	}
}

// TestTypeDeclarationPerScope covers same-named fields arriving from
// different engine prefixes, which is what deriving sub-engines with
// WithPrefix produces.
//
// The dedup used to compare the bare field name with the scope discarded, so
// only the first scope to emit "hits" got a "# TYPE" line and every later one
// ingested as untyped.
func TestTypeDeclarationPerScope(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{}

	scopes := []string{"alpha", "beta", "gamma"}
	for _, scope := range scopes {
		handler.HandleMeasures(now, stats.Measure{
			Name: scope,
			Fields: []stats.Field{
				stats.MakeField("hits", 1, stats.Counter),
				stats.MakeField("size", 2, stats.Gauge),
			},
		})
	}

	var buf strings.Builder
	handler.WriteStats(&buf)
	out := buf.String()

	for _, scope := range scopes {
		for _, want := range []string{
			"# TYPE " + scope + "_hits counter",
			"# TYPE " + scope + "_size gauge",
		} {
			if n := strings.Count(out, want); n != 1 {
				t.Errorf("found %q %d times, expected exactly 1:\n%s", want, n, out)
			}
		}
	}

	// Six metrics, six type declarations, none repeated.
	if n := strings.Count(out, "# TYPE "); n != 2*len(scopes) {
		t.Errorf("found %d type declarations, expected %d", n, 2*len(scopes))
	}
}

// TestTypeDeclarationAcrossAdjacentScopes is the tighter version of the case
// above: when each scope exposes the same single field name, the families are
// adjacent in the output and a dedup that compares the bare name suppresses
// every one after the first.
func TestTypeDeclarationAcrossAdjacentScopes(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{}

	scopes := []string{"alpha", "beta", "gamma"}
	for _, scope := range scopes {
		handler.HandleMeasures(now, stats.Measure{
			Name: scope,
			Fields: []stats.Field{
				stats.MakeField("hits", 1, stats.Counter),
				// A histogram spans three series names, so its family only
				// stays contiguous if the sort orders by scope before name.
				// Sorting on the bare name groups every scope's _bucket
				// together and pushes _count and _sum away from it, which
				// makes the same family declare its type more than once.
				stats.MakeField("latency", 0.1, stats.Histogram),
			},
		})
	}

	var buf strings.Builder
	handler.WriteStats(&buf)
	out := buf.String()

	for _, scope := range scopes {
		for _, want := range []string{
			"# TYPE " + scope + "_hits counter",
			"# TYPE " + scope + "_latency histogram",
		} {
			if n := strings.Count(out, want); n != 1 {
				t.Errorf("found %q %d times, expected exactly 1:\n%s", want, n, out)
			}
		}
	}

	if n := strings.Count(out, "# TYPE "); n != 2*len(scopes) {
		t.Errorf("found %d type declarations, expected %d:\n%s", n, 2*len(scopes), out)
	}
}
