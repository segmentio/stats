package httpstats

import (
	"bufio"
	"net"
	"net/http"

	"github.com/segmentio/stats"
)

func NewHandler(client stats.Client, handler http.Handler) http.Handler {
	return httpHandler{
		handler: handler,
		metrics: NewServerMetrics(client),
	}
}

type httpHandler struct {
	handler http.Handler
	metrics *Metrics
}

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var clock stats.Clock
	clock, req.Body = h.metrics.ObserveRequest(req)

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
	clock   stats.Clock
	metrics *Metrics
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
	if w.metrics != nil {
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
		w.metrics.Responses.Count.Add(1, tags...)
		w.metrics.Responses.HeaderSizes.Observe(float64(len(res.Header)), tags...)
		w.metrics.Responses.HeaderBytes.Observe(float64(responseHeaderLength(res)), tags...)
		w.metrics.Responses.BodyBytes.Observe(float64(w.bytes), tags...)
		w.clock.Stop(tags...)

		// release resources
		w.metrics = nil
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
