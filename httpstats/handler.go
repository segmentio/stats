package httpstats

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/segmentio/stats"
)

// NewHandler wraps h to produce metrics on eng for every request received and
// every response sent.
func NewHandler(eng *stats.Engine, h http.Handler, tags ...stats.Tag) http.Handler {
	return &handler{
		handler: h,
		metrics: MakeServerMetrics(eng, tags...),
	}
}

type handler struct {
	handler http.Handler
	metrics Metrics
}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	start := time.Now()

	req.Body, _ = h.metrics.ObserveRequest(req)

	w := &httpResponseWriter{
		ResponseWriter: res,
		req:            req,
		start:          start,
		metrics:        h.metrics,
	}

	h.handler.ServeHTTP(w, req)
	w.complete()
}

type httpResponseWriter struct {
	http.ResponseWriter
	header  http.Header
	bytes   int
	status  int
	metrics Metrics
	start   time.Time
	req     *http.Request
}

func (w *httpResponseWriter) WriteHeader(status int) {
	if w.header == nil {
		w.status = status
		w.header = copyHeader(w.Header())
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *httpResponseWriter) Write(b []byte) (n int, err error) {
	w.WriteHeader(http.StatusOK)

	if n, err = w.ResponseWriter.Write(b); n > 0 {
		w.bytes += n
	}

	return
}

func (w *httpResponseWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	if conn, buf, err = w.ResponseWriter.(http.Hijacker).Hijack(); err == nil {
		w.complete()
	}
	return
}

func (w *httpResponseWriter) complete() {
	if w.header != nil {
		w.req.Body.Close()

		now := time.Now()
		res := &http.Response{
			ProtoMajor:    w.req.ProtoMajor,
			ProtoMinor:    w.req.ProtoMinor,
			StatusCode:    w.status,
			Header:        w.header,
			Request:       w.req,
			ContentLength: -1,
		}

		tags := make([]stats.Tag, 0, len(w.metrics.resTags)+20)
		tags = append(tags, w.metrics.resTags...)
		tags = appendResponseTags(tags, res)
		tags = appendRequestTags(tags, res.Request)

		rawTags := stats.MakeRawTags(tags)

		w.metrics.incrMessageCounter(tags, rawTags, now)
		w.metrics.observeHeaderSize(len(res.Header), tags, rawTags, now)
		w.metrics.observeHeaderLength(responseHeaderLength(res), tags, rawTags, now)
		w.metrics.observeBodyLength(w.bytes, tags, rawTags, now)
		w.metrics.observeRTT(now.Sub(w.start), tags, rawTags, now)
	}
}
