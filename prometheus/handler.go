package prometheus

import (
	"compress/gzip"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/segmentio/stats"
)

// Handler is a type that bridges the stats API to a prometheus-compatible HTTP
// endpoint.
//
// Typically, a program creates one Handler, registers it to the stats package,
// and adds it to the muxer used by the application under the /metrics path.
type Handler struct {
	// MetricTimeout defines how long the handler exposes metrics that aren't
	// receiving updates.
	//
	// The default is to use a 2 minutes metric timeout.
	MetricTimeout time.Duration

	opcount uint64
	metrics metricStore
}

// HandleMetric satisfies the stats.Handler interface.
func (h *Handler) HandleMetric(m *stats.Metric) {
	mtime := m.Time
	if mtime.IsZero() {
		mtime = time.Now()
	}

	labels := (labels{}).appendTags(m.Tags...)
	sort.Sort(labels)

	h.metrics.update(metric{
		mtype:  metricTypeOf(m.Type),
		name:   m.Namespace + "_" + m.Name,
		help:   "",
		value:  m.Value,
		time:   mtime,
		labels: labels,
	})

	// Every 10K updates we cleanup the metric store of outdated entries to
	// having memory leaks if the program has generated metrics for a pair of
	// metric name and labels that won't be seen again.
	if (atomic.AddUint64(&h.opcount, 1) % 10000) == 0 {
		h.cleanup()
	}
}

func (h *Handler) cleanup() {
	// TODO:
}

func (h *Handler) timeout() time.Duration {
	if timeout := h.MetricTimeout; timeout != 0 {
		return timeout
	}
	return 2 * time.Minute
}

// ServeHTTP satsifies the http.Handler interface.
func (h *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	metrics := h.metrics.collect(make([]metric, 0, 10000))

	w := io.Writer(res)
	res.Header().Set("Content-Type", "text/plain; version=0.0.4")

	if acceptEncoding(req.Header.Get("Accept-Endoing"), "gzip") {
		res.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		defer zw.Close()
		w = zw
	}

	b := make([]byte, 1024)

	for _, m := range metrics {
		b = appendMetric(b[:0], m)
		b = append(b, '\n')
		w.Write(b)
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
