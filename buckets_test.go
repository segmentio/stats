package stats_test

import (
	"strings"
	"testing"
	"time"

	stats "github.com/segmentio/stats/v5"
	_ "github.com/segmentio/stats/v5/httpstats"
	"github.com/segmentio/stats/v5/prometheus"
	"github.com/segmentio/stats/v5/statstest"
)

// TestEngineSetBucketsKeyMatchesObserve pins the invariant SetBuckets exists
// for: the registry key it writes is exactly the one Observe produces for the
// same name, on the engine it was called on and on any sub-engine derived
// from it.
//
// The key is read back from what Observe actually emitted rather than being
// restated here, so the test fails if either side of the pair changes.
func TestEngineSetBucketsKeyMatchesObserve(t *testing.T) {
	for _, test := range []struct {
		scenario string
		name     string
		engine   func(stats.Handler) *stats.Engine
	}{
		{
			scenario: "engine with a prefix",
			name:     "latency",
			engine:   func(h stats.Handler) *stats.Engine { return stats.NewEngine("app", h) },
		},
		{
			scenario: "engine with no prefix",
			name:     "latency",
			engine:   func(h stats.Handler) *stats.Engine { return stats.NewEngine("", h) },
		},
		{
			scenario: "sub-engine derived with WithPrefix",
			name:     "latency",
			engine: func(h stats.Handler) *stats.Engine {
				return stats.NewEngine("app", h).WithPrefix("sub")
			},
		},
		{
			scenario: "sub-engine derived twice",
			name:     "latency",
			engine: func(h stats.Handler) *stats.Engine {
				return stats.NewEngine("app", h).WithPrefix("sub").WithPrefix("deeper")
			},
		},
		{
			// Observe splits the name on its last dot and prefixes only the
			// measure half, so SetBuckets has to do the same.
			scenario: "dotted name",
			name:     "db.latency",
			engine:   func(h stats.Handler) *stats.Engine { return stats.NewEngine("app", h) },
		},
	} {
		t.Run(test.scenario, func(t *testing.T) {
			name := test.name

			h := &statstest.Handler{}
			e := test.engine(h)

			e.SetBuckets(name, 0.1, 0.2, 0.3)
			e.Observe(name, 0.15)

			// Find the measure Observe produced, skipping the go_version
			// gauge the engine reports once.
			var key stats.Key
			var found bool
			for _, m := range h.Measures() {
				for _, f := range m.Fields {
					if strings.HasSuffix(name, f.Name) {
						key = stats.Key{Measure: m.Name, Field: f.Name}
						found = true
					}
				}
			}
			if !found {
				t.Fatalf("Observe produced no measure for %q", name)
			}

			buckets, ok := stats.Buckets[key]
			if !ok {
				t.Fatalf("SetBuckets did not register %#v; registry holds %#v",
					key, keysOf(stats.Buckets))
			}
			if len(buckets) != 3 {
				t.Errorf("registered %d buckets, expected 3", len(buckets))
			}

			delete(stats.Buckets, key)
		})
	}
}

// TestEngineSetBucketsEndToEnd runs the whole chain: buckets registered on a
// sub-engine reach the exposition, rather than the metric silently falling
// back to the default set.
func TestEngineSetBucketsEndToEnd(t *testing.T) {
	ph := &prometheus.Handler{}
	e := stats.NewEngine("svc", ph).WithPrefix("sub")

	e.SetBuckets("latency", 0.1, 0.2, 0.3)
	defer delete(stats.Buckets, stats.Key{Measure: "svc.sub", Field: "latency"})

	e.Observe("latency", 0.15)

	var buf strings.Builder
	ph.WriteStats(&buf)
	out := buf.String()

	for _, want := range []string{
		`svc_sub_latency_bucket{le="0.1"} 0`,
		`svc_sub_latency_bucket{le="0.2"} 1`,
		`svc_sub_latency_bucket{le="0.3"} 1`,
		`svc_sub_latency_bucket{le="+Inf"} 1`,
		`svc_sub_latency_count 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}

	// Three registered boundaries plus +Inf. More would mean the registration
	// missed and DefaultBuckets was used instead.
	if n := strings.Count(out, "svc_sub_latency_bucket{"); n != 4 {
		t.Errorf("found %d bucket series, expected 4:\n%s", n, out)
	}
}

// TestBucketsSetStillWorks covers the pre-existing registration path, which
// SetBuckets is additive to.
func TestBucketsSetStillWorks(t *testing.T) {
	key := stats.Key{Measure: "legacy.svc", Field: "latency"}
	defer delete(stats.Buckets, key)

	stats.Buckets.Set("legacy.svc.latency", 0.1, 0.2)

	if _, ok := stats.Buckets[key]; !ok {
		t.Errorf("Buckets.Set did not register %#v", key)
	}
}

func keysOf(b stats.HistogramBuckets) []stats.Key {
	keys := make([]stats.Key, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	return keys
}

// TestHistogramBucketsSetKeyDottedField covers the key Set cannot express.
// Set splits its argument on the last ".", so a field carrying one of its own
// is unreachable through it — and every histogram httpstats reports has such
// a field.
func TestHistogramBucketsSetKeyDottedField(t *testing.T) {
	key := stats.Key{Measure: "http.message", Field: "body.bytes"}

	b := stats.HistogramBuckets{}
	b.SetKey(key, 100, 1000)

	if _, ok := b[key]; !ok {
		t.Fatalf("SetKey did not register %#v; registry holds %#v", key, keysOf(b))
	}

	legacy := stats.HistogramBuckets{}
	legacy.Set("http.message.body.bytes", 100, 1000)

	if _, ok := legacy[key]; ok {
		t.Error("Set reached the dotted-field key; SetKey would be unnecessary")
	}
}

// TestHistogramBucketsLookupUnprefixed pins the resolution a package doing its
// registration from init() depends on: it cannot know the prefix an engine
// will add, so the name it registers has to match a measure carrying one.
func TestHistogramBucketsLookupUnprefixed(t *testing.T) {
	b := stats.HistogramBuckets{}
	b.SetUnprefixed(stats.Key{Measure: "http.message", Field: "body.bytes"}, 100, 1000)

	for _, measure := range []string{
		"http.message",
		"myapp.http.message",
		"myapp.sub.http.message",
	} {
		if v := b.Lookup(measure, "body.bytes"); len(v) != 2 {
			t.Errorf("Lookup(%q) resolved %d buckets, expected 2", measure, len(v))
		}
	}

	if v := b.Lookup("myapp.http.message", "header.bytes"); v != nil {
		t.Error("Lookup matched a different field")
	}
	if v := b.Lookup("message", "body.bytes"); v != nil {
		t.Error("Lookup matched half a segment")
	}
}

// TestHistogramBucketsLookupExactWins keeps a program able to override what a
// package registered for the same measure.
func TestHistogramBucketsLookupExactWins(t *testing.T) {
	b := stats.HistogramBuckets{}
	b.SetUnprefixed(stats.Key{Measure: "conn.read", Field: "bytes"}, 1, 2, 3)
	b.SetKey(stats.Key{Measure: "myapp.conn.read", Field: "bytes"}, 9)

	if v := b.Lookup("myapp.conn.read", "bytes"); len(v) != 1 {
		t.Errorf("exact registration lost to the unprefixed one (%d buckets)", len(v))
	}
}

// TestHistogramBucketsSuffixMatchingIsOptIn is why SetUnprefixed exists as a
// separate call. Matching by suffix cannot tell a derived measure from an
// unrelated one ending the same way, so only registrations that ask for it
// take part: svc.billing must not inherit a set registered for billing.
func TestHistogramBucketsSuffixMatchingIsOptIn(t *testing.T) {
	b := stats.HistogramBuckets{}
	b.SetKey(stats.Key{Measure: "billing", Field: "size"}, 0.25, 0.75)

	if v := b.Lookup("svc.billing", "size"); v != nil {
		t.Errorf("a SetKey registration was matched by suffix: %v", v)
	}
}

// TestEngineSetBucketsFromAncestor pins what the doc comment promises: one
// init function on the root engine covers the whole tree, by naming the path
// to each metric rather than holding a reference to every sub-engine.
func TestEngineSetBucketsFromAncestor(t *testing.T) {
	ph := &prometheus.Handler{}
	root := stats.NewEngine("anc", ph)

	root.SetBuckets("db.latency", 0.2, 0.4)
	defer delete(stats.Buckets, stats.Key{Measure: "anc.db", Field: "latency"})

	root.WithPrefix("db").Observe("latency", 0.3)

	var buf strings.Builder
	ph.WriteStats(&buf)
	out := buf.String()

	// Two registered boundaries plus +Inf. Eleven would mean the registration
	// missed and DefaultBuckets was used.
	if n := strings.Count(out, "anc_db_latency_bucket{"); n != 3 {
		t.Errorf("found %d bucket series, expected 3:\n%s", n, out)
	}
}

// TestHTTPStatsBucketsReachTheHandler is the regression this whole change is
// for. httpstats registers byte boundaries for its message histograms; until
// they resolved, every one of them took DefaultBuckets instead — eleven
// boundaries between 0.005 and 10 seconds, none of which a byte count can
// reach, so +Inf held every observation.
func TestHTTPStatsBucketsReachTheHandler(t *testing.T) {
	ph := &prometheus.Handler{}

	ph.HandleMeasures(time.Now(), stats.Measure{
		Name:   "myapp.http.message",
		Fields: []stats.Field{stats.MakeField("body.bytes", 5000, stats.Histogram)},
	})

	var buf strings.Builder
	ph.WriteStats(&buf)
	out := buf.String()

	if !strings.Contains(out, `le="10000"`) {
		t.Errorf("registered byte boundaries missing:\n%s", out)
	}
	if strings.Contains(out, `le="0.005"`) {
		t.Errorf("fell back to DefaultBuckets:\n%s", out)
	}
}
