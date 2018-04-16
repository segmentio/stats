package prometheus

import (
	"compress/gzip"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/stats"
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
	// Note that triming only applies to the metric namespace, the metric
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
	Buckets stats.HistogramBuckets

	opcount uint64
	metrics metricStore
}

// HandleMetric satisfies the stats.Handler interface.
func (h *Handler) HandleMeasures(mtime time.Time, measures ...stats.Measure) {
	cache := handleMetricPool.Get().(*handleMetricCache)

	for _, m := range measures {
		scope := h.trimPrefix(m.Name)

		cache.labels = cache.labels[:0]
		cache.labels = cache.labels.appendTags(m.Tags...)

		for _, f := range m.Fields {
			var buckets []stats.Value
			var mtype = typeOf(f.Type())

			if mtype == histogram {
				k := stats.Key{Measure: m.Name, Field: f.Name}

				if b := h.Buckets; b != nil {
					buckets = b[k]
				} else {
					buckets = stats.Buckets[k]
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
	if (atomic.AddUint64(&h.opcount, 1) % 10000) == 0 {
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
// It will not call flush. Make sure the Close and Flush are handled at the caller
func (h *Handler) WriteStats(w io.Writer) {
	b := make([]byte, 1024)

	var lastMetricName string
	metrics := h.metrics.collect(make([]metric, 0, 10000))
	sort.Sort(byNameAndLabels(metrics))

	for i, m := range metrics {
		b = b[:0]
		name := m.rootName()

		if name == lastMetricName {
			// Silence the repeated output of type for values belonging to the
			// same metric.
			m.mtype, m.help = untyped, ""
		} else if i != 0 {
			// After every metric we want to output an empty line to make the
			// output easier to read.
			b = append(b, '\n')
		}

		w.Write(appendMetric(b, m))
		lastMetricName = name
	}
}

func acceptEncoding(accept string, check string) bool {
	for _, coding := range strings.Split(accept, ",") {
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
	New: func() interface{} {
		return &handleMetricCache{labels: make(labels, 0, 8)}
	},
}

func (cache *handleMetricCache) Len() int {
	return len(cache.labels)
}

func (cache *handleMetricCache) Swap(i int, j int) {
	cache.labels[i], cache.labels[j] = cache.labels[j], cache.labels[i]
}

func (cache *handleMetricCache) Less(i int, j int) bool {
	return cache.labels[i].less(cache.labels[j])
}

// DefaultHandler is a prometheus handler configured to trim the default metric
// namespace off of metrics that it handles.
var DefaultHandler = &Handler{
	TrimPrefix: stats.DefaultEngine.Prefix,
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
