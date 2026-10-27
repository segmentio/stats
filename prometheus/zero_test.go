package prometheus

import (
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
)

// setNow points the handler's wall clock at *clock for the rest of the test.
func setNow(t *testing.T, clock *time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return *clock }
	t.Cleanup(func() { now = prev })
}

// seriesKey identifies a collected sample the way a scraper does: by its
// exposed name and its labels.
func seriesKey(m metric) string {
	var b strings.Builder
	b.WriteString(m.scope + "_" + m.name)
	for _, l := range m.labels {
		b.WriteString("," + l.name + "=" + l.value)
	}
	return b.String()
}

// promIncrease is what increase() over the whole run computes for one series:
// the sum of the changes between consecutive samples, treating a decrease as
// a reset to zero. The first sample is never counted.
func promIncrease(samples []float64) float64 {
	var total float64
	for i := 1; i < len(samples); i++ {
		if d := samples[i] - samples[i-1]; d >= 0 {
			total += d
		} else {
			total += samples[i]
		}
	}
	return total
}

// scrapeCounts runs a handler for 30 minutes of simulated time, scraping it
// every 15s, and returns what increase() sums to over every counter and
// histogram _count series, along with the number of increments made.
//
// One series is busy and never expires. Three others, one per workspace, see
// a short burst every few minutes and are expired by MetricTimeout in between
// — the shape of a counter labeled by a per-customer ID.
func scrapeCounts(t *testing.T, zeroPeriod time.Duration) (got, want float64) {
	const (
		scrapeInterval = 15 * time.Second
		duration       = 30 * time.Minute
	)

	clock := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	setNow(t, &clock)
	start := clock

	h := &Handler{ZeroPeriod: zeroPeriod, Buckets: stats.HistogramBuckets{}}
	samples := map[string][]float64{}

	scrape := func() {
		for _, m := range h.metrics.collect(nil, clock) {
			if strings.HasSuffix(m.name, "_total") || strings.HasSuffix(m.name, "_count") {
				k := seriesKey(m)
				samples[k] = append(samples[k], m.value)
			}
		}
	}

	for step := time.Duration(0); step < duration; step += time.Second {
		clock = start.Add(step)

		measures := []stats.Measure{{
			Name:   "svc.requests",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
		}}
		for i, ws := range []string{"ws1", "ws2", "ws3"} {
			if step%(time.Duration(4+i)*time.Minute) < 3*time.Second {
				measures = append(measures, stats.Measure{
					Name: "svc.responses",
					Fields: []stats.Field{
						stats.MakeField("count", 1, stats.Counter),
						stats.MakeField("rtt", 0.1, stats.Histogram),
					},
					Tags: []stats.Tag{stats.T("workspace_id", ws)},
				})
			}
		}
		for _, m := range measures {
			want += float64(len(m.Fields))
			h.HandleMeasures(clock, m)
		}

		if step%scrapeInterval == 0 {
			// Expire as eagerly as the handler is allowed to, rather than
			// every 10K updates, so the result does not depend on traffic.
			h.metrics.cleanup(clock.Add(-h.timeout()))
			scrape()
		}
	}

	// Give the last series created time to leave its zero period.
	clock = clock.Add(h.zeroPeriod() + scrapeInterval)
	scrape()

	for _, s := range samples {
		got += promIncrease(s)
	}
	return got, want
}

// TestZeroPeriodCountsEveryIncrement reproduces the undercount: increase()
// never counts a series' first sample, so sparse series that MetricTimeout
// keeps expiring and re-creating lose counts each time they come back.
func TestZeroPeriodCountsEveryIncrement(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		got, want := scrapeCounts(t, -1)
		if got >= want {
			t.Fatalf("increase() = %v of %v; the simulation no longer reproduces the loss", got, want)
		}
		t.Logf("increase() = %v of %v without a zero period", got, want)
	})

	t.Run("default", func(t *testing.T) {
		if got, want := scrapeCounts(t, 0); got != want {
			t.Fatalf("increase() = %v, expected %v", got, want)
		}
	})
}

// TestZeroPeriodIsTimeBased checks that every scrape during the zero period
// sees 0, however many there are, so that two scrapers of one endpoint both
// see the series start from zero.
func TestZeroPeriodIsTimeBased(t *testing.T) {
	clock := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	setNow(t, &clock)

	h := &Handler{}
	h.HandleMeasures(clock, stats.Measure{
		Name: "svc",
		Fields: []stats.Field{
			stats.MakeField("hits", 3, stats.Counter),
			stats.MakeField("rtt", 0.1, stats.Histogram),
			stats.MakeField("level", 7, stats.Gauge),
		},
	})

	write := func() string {
		var buf strings.Builder
		h.WriteStats(&buf)
		return buf.String()
	}

	for range 3 {
		out := write()
		for _, want := range []string{
			"svc_hits_total 0",
			`svc_rtt_bucket{le="+Inf"} 0`,
			"svc_rtt_count 0",
			"svc_rtt_sum 0",
			// Gauges carry no history for a scraper to difference.
			"svc_level 7",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q during the zero period:\n%s", want, out)
			}
		}
	}

	clock = clock.Add(h.zeroPeriod())
	out := write()
	for _, want := range []string{
		"svc_hits_total 3",
		`svc_rtt_bucket{le="+Inf"} 1`,
		"svc_rtt_count 1",
		"svc_rtt_sum 0.1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q after the zero period:\n%s", want, out)
		}
	}
}

// TestZeroPeriodOutlivesExpiry checks that a state is not expired while it
// publishes zero, nor until it has published its real value for a full
// MetricTimeout, even when it receives no further updates.
func TestZeroPeriodOutlivesExpiry(t *testing.T) {
	clock := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	setNow(t, &clock)
	created := clock

	h := &Handler{MetricTimeout: 30 * time.Second}
	h.HandleMeasures(clock, stats.Measure{
		Name:   "svc",
		Fields: []stats.Field{stats.MakeField("hits", 1, stats.Counter)},
	})

	live := func() bool {
		h.metrics.cleanup(clock.Add(-h.timeout()))
		return len(h.metrics.collect(nil, clock)) != 0
	}

	// Past MetricTimeout since the update, but still in the zero period.
	clock = created.Add(45 * time.Second)
	if !live() {
		t.Fatal("expired during its zero period")
	}

	// Value published since created+ZeroPeriod; MetricTimeout of that has not
	// passed.
	clock = created.Add(h.zeroPeriod() + 29*time.Second)
	if !live() {
		t.Fatal("expired before its value was published for a full MetricTimeout")
	}

	clock = created.Add(h.zeroPeriod() + 31*time.Second)
	if live() {
		t.Fatal("not expired a MetricTimeout after its zero period")
	}
}
