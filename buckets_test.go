package stats_test

import (
	"strings"
	"testing"

	stats "github.com/segmentio/stats/v5"
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
