package httpstats

import (
	"bufio"
	"net"
	"net/http"

	"github.com/segmentio/stats"
)

func NewHandler(handler http.Handler, eng *stats.Engine, tags ...stats.Tag) http.Handler {
	return &httpHandler{
		handler: handler,
		metrics: MakeServerMetrics(eng, tags...),
	}
}

type httpHandler struct {
	handler http.Handler
	metrics Metrics
}

func (h *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	clock := h.metrics.RTT.Start()

	req.Body, _ = h.metrics.ObserveRequest(req)

	w := &httpResponseWriter{
		ResponseWriter: res,
		req:            req,
		clock:          clock,
		metrics:        h.metrics,
	}
	res = w

	if _, ok := w.ResponseWriter.(http.Hijacker); ok {
		res = httpResponseHijacker{w}
	}

	h.handler.ServeHTTP(res, req)
	w.complete()
}

type httpResponseWriter struct {
	http.ResponseWriter
	header  http.Header
	bytes   int
	status  int
	metrics Metrics
	clock   *stats.Clock
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

func (w *httpResponseWriter) complete() {
	if w.header != nil {
		// make sure the request body was closed
		w.req.Body.Close()

		// report response metrics
		res := &http.Response{
			ProtoMajor:    w.req.ProtoMajor,
			ProtoMinor:    w.req.ProtoMinor,
			StatusCode:    w.status,
			Header:        w.header,
			Request:       w.req,
			ContentLength: -1,
		}

		tags := makeResponseTags(res)
		w.metrics.Responses.Count.Clone(tags...).Incr()
		w.metrics.Responses.HeaderSizes.Clone(tags...).Observe(float64(len(res.Header)))
		w.metrics.Responses.HeaderBytes.Clone(tags...).Observe(float64(responseHeaderLength(res)))
		w.metrics.Responses.BodyBytes.Clone(tags...).Observe(float64(w.bytes))
		w.clock.Clone(tags...).Stop()

		// release resources
		w.header = nil
		w.clock = nil
		w.req = nil
	}
}

type httpResponseHijacker struct {
	*httpResponseWriter
}

func (w httpResponseHijacker) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	if conn, buf, err = w.ResponseWriter.(http.Hijacker).Hijack(); err == nil {
		w.complete()
	}
	return
}
