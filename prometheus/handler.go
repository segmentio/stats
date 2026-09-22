package prometheus

import (
	"compress/gzip"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/stats/v5"
)

// Handler is a type that bridges the stats API to a prometheus-compatible HTTP
// endpoint.
//
// Typically, a program creates one Handler, registers it to the stats package,
// and adds it to the muxer used by the application under the /metrics path.
//
// The handle ignores histograms that have no buckets set.
type Handler struct {
	// Setting this field will trim this prefix from metric namespaces of the
	// metrics received by this handler.
	//
	// Unlike statsd-like systems, it is common for prometheus metrics to not
	// be prefixed and instead use labels to identify which service or group
	// of services the metrics are coming from. The intent of this field is to
	// provide support for this use case.
	//
	// Note that trimming only applies to the metric namespace, the metric
	// name will always be left untouched.
	//
	// If empty, no prefix trimming is done.
	TrimPrefix string

	// MetricTimeout defines how long the handler exposes metrics that aren't
	// receiving updates.
	//
	// The default is to use a 2 minutes metric timeout.
	MetricTimeout time.Duration

	// Buckets is the registry of histogram buckets used by the handler,
	// If nil, stats.Buckets is used instead.
	//
	// Histograms with no entry in the registry fall back to DefaultBuckets.
	Buckets stats.HistogramBuckets

	opcount atomic.Uint64
	metrics metricStore
}

// HandleMeasures satisfies the stats.Handler interface.
func (h *Handler) HandleMeasures(mtime time.Time, measures ...stats.Measure) {
	cache := handleMetricPool.Get().(*handleMetricCache)

	for _, m := range measures {
		scope := h.trimPrefix(m.Name)

		cache.labels = cache.labels[:0]
		cache.labels = cache.labels.appendTags(m.Tags...)

		for _, f := range m.Fields {
			var buckets []stats.Value
			mtype := typeOf(f.Type())

			if mtype == histogram {
				k := stats.Key{Measure: m.Name, Field: f.Name}

				if b := h.Buckets; b != nil {
					buckets = b[k]
				} else {
					buckets = stats.Buckets[k]
				}

				// A registry miss returns a nil slice with no error, which
				// used to mean the histogram was published with _sum and
				// _count but no _bucket series at all — nothing looked wrong,
				// and no percentile could be computed. Fall back so that a
				// histogram is never silently bucket-less.
				if len(buckets) == 0 {
					buckets = DefaultBuckets
				}

				// makeMetricBuckets appends the +Inf bucket itself. Ending a
				// registered set with math.Inf(+1) is a common idiom — every
				// bucket set in httpstats, netstats and procstats does it —
				// and would otherwise yield two le="+Inf" series, the second
				// of them unreachable because metricBuckets.update stops at
				// the first match.
				//
				// Trimming here rather than in makeMetricBuckets is
				// deliberate: metricState.update decides whether to rebuild by
				// comparing against len(buckets)+1, so a makeMetricBuckets
				// that sometimes returned len(buckets) entries would rebuild —
				// and zero the counts — on every observation.
				if n := len(buckets); n > 0 && math.IsInf(valueOf(buckets[n-1]), 1) {
					buckets = buckets[:n-1]
				}
			}

			h.metrics.update(metric{
				mtype:  mtype,
				scope:  scope,
				name:   f.Name,
				value:  valueOf(f.Value),
				time:   mtime,
				labels: cache.labels,
			}, buckets)
		}

		for i := range cache.labels {
			cache.labels[i] = label{}
		}
	}

	handleMetricPool.Put(cache)

	// Every 10K updates we cleanup the metric store of outdated entries to
	// having memory leaks if the program has generated metrics for a pair of
	// metric name and labels that won't be seen again.
	if (h.opcount.Add(1) % 10000) == 0 {
		h.metrics.cleanup(time.Now().Add(-h.timeout()))
	}
}

func (h *Handler) trimPrefix(s string) string {
	s = strings.TrimPrefix(s, h.TrimPrefix)
	if len(s) != 0 && s[0] == '.' {
		s = s[1:]
	}
	return s
}

func (h *Handler) timeout() time.Duration {
	if timeout := h.MetricTimeout; timeout != 0 {
		return timeout
	}
	return 2 * time.Minute
}

// ServeHTTP satisfies the http.Handler interface.
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET", "HEAD":
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w := io.Writer(res)
	res.Header().Set("Content-Type", "text/plain; version=0.0.4")

	if acceptEncoding(req.Header.Get("Accept-Encoding"), "gzip") {
		res.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		defer zw.Close()
		w = zw
	}

	h.WriteStats(w)
}

// WriteStats accepts a writer and pushes metrics (one at a time) to it.
// An example could be if you just want to print all the metrics on to Stdout
// It will not call flush. Make sure the Close and Flush are handled at the caller.
func (h *Handler) WriteStats(w io.Writer) {
	b := make([]byte, 1024)

	// A metric family is identified by its scope and root name together. The
	// scope cannot be dropped here: two sub-engines derived with WithPrefix
	// commonly expose the same field name, and comparing the bare name made
	// the second family look like a repeat of the first, so its "# TYPE" line
	// was suppressed and it ingested as untyped.
	//
	// byNameAndLabels.Less orders by scope before name for the same reason.
	//
	// Tracking every family declared so far, rather than comparing against the
	// previous metric, is what makes "declared exactly once" hold. The sort
	// keeps a scope contiguous but not a root name within it, because it
	// orders on the series name while a family is keyed on the root: a
	// histogram "q" emits q_bucket, q_count and q_sum, and a sibling "q_bytes"
	// sorts between the first two. A one-metric memory forgets q was declared
	// and declares it again — and a repeated declaration is not a dropped
	// sample, it makes the text format parser reject the whole exposition, so
	// the entire scrape fails.
	declared := make(map[metricKey]struct{})

	metrics := h.metrics.collect(make([]metric, 0, 10000))
	sort.Sort(byNameAndLabels(metrics))

	for i, m := range metrics {
		b = b[:0]
		family := metricKey{scope: m.scope, name: m.rootName()}

		if _, seen := declared[family]; seen {
			// Silence the repeated output of type for values belonging to the
			// same metric.
			m.mtype, m.help = untyped, ""
		} else if i != 0 {
			// After every metric we want to output an empty line to make the
			// output easier to read.
			b = append(b, '\n')
		}

		_, _ = w.Write(appendMetric(b, m))
		declared[family] = struct{}{}
	}
}

func acceptEncoding(accept, check string) bool {
	for coding := range strings.SplitSeq(accept, ",") {
		if coding = strings.TrimSpace(coding); strings.HasPrefix(coding, check) {
			return true
		}
	}
	return false
}

type handleMetricCache struct {
	labels labels
}

var handleMetricPool = sync.Pool{
	New: func() any {
		return &handleMetricCache{labels: make(labels, 0, 8)}
	},
}

func (cache *handleMetricCache) Len() int {
	return len(cache.labels)
}

func (cache *handleMetricCache) Swap(i, j int) {
	cache.labels[i], cache.labels[j] = cache.labels[j], cache.labels[i]
}

func (cache *handleMetricCache) Less(i, j int) bool {
	return cache.labels[i].less(cache.labels[j])
}

// DefaultHandler is a prometheus handler configured to trim the default metric
// namespace off of metrics that it handles.
var DefaultHandler = &Handler{
	TrimPrefix: stats.DefaultEngine.Prefix,
}

// DefaultBuckets is the bucket set used for histograms that have no boundaries
// registered in stats.Buckets or in Handler.Buckets.
//
// The boundaries are the ones used by the reference Prometheus client, chosen
// for request latencies measured in seconds. stats.Duration values are
// converted to seconds before bucketing, so timing histograms land on this
// range without configuration.
//
// They are a starting point, not a substitute for choosing boundaries: a
// bucketed percentile is only as accurate as the bucket it falls in, and a
// histogram whose values sit outside this range lands entirely in the +Inf
// bucket. Register real boundaries with Engine.SetBuckets wherever p99
// accuracy matters.
//
// Programs may replace this during initialization, before any measure is
// handled.
var DefaultBuckets = []stats.Value{
	stats.ValueOf(0.005),
	stats.ValueOf(0.01),
	stats.ValueOf(0.025),
	stats.ValueOf(0.05),
	stats.ValueOf(0.1),
	stats.ValueOf(0.25),
	stats.ValueOf(0.5),
	stats.ValueOf(1.0),
	stats.ValueOf(2.5),
	stats.ValueOf(5.0),
	stats.ValueOf(10.0),
}

func typeOf(t stats.FieldType) metricType {
	switch t {
	case stats.Counter:
		return counter
	case stats.Gauge:
		return gauge
	case stats.Histogram:
		return histogram
	default:
		return untyped
	}
}

func valueOf(v stats.Value) float64 {
	switch v.Type() {
	case stats.Bool:
		if v.Bool() {
			return 1.0
		}
	case stats.Int:
		return float64(v.Int())
	case stats.Uint:
		return float64(v.Uint())
	case stats.Float:
		return v.Float()
	case stats.Duration:
		return v.Duration().Seconds()
	}
	return 0.0
}
