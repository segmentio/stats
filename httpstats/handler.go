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
		handler:  h,
		inflight: stats.MakeGauge(eng, "http.inflight", tags...),
		metrics:  MakeServerMetrics(eng, tags...),
	}
}

type handler struct {
	handler  http.Handler
	inflight stats.Gauge
	metrics  Metrics
}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	h.inflight.Incr()
	defer h.inflight.Decr()

	start := time.Now()
	body, _ := h.metrics.ObserveRequest(req)

	w := &httpResponseWriter{
		ResponseWriter: res,
		req:            req,
		start:          start,
		metrics:        h.metrics,
		body:           body.(*messageBody),
	}
	defer w.complete()

	req.Body = body
	h.handler.ServeHTTP(w, req)
}

type httpResponseWriter struct {
	http.ResponseWriter
	bytes       int
	status      int
	metrics     Metrics
	start       time.Time
	req         *http.Request
	body        *messageBody
	wroteHeader bool
}

func (w *httpResponseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = status
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
		w.wroteHeader = true
		w.complete()
	}
	return
}

func (w *httpResponseWriter) complete() {
	w.WriteHeader(http.StatusOK)
	w.body.complete()

	now := time.Now()
	res := &http.Response{
		ProtoMajor:    w.req.ProtoMajor,
		ProtoMinor:    w.req.ProtoMinor,
		StatusCode:    w.status,
		Header:        w.Header(),
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
