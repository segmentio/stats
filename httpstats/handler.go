package httpstats

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/segmentio/stats"
)

// NewHandler wraps h to produce metrics on the default engine for every request
// received and every response sent.
func NewHandler(h http.Handler) http.Handler {
	return NewHandlerWith(nil, h)
}

// NewHandlerWith wraps h to produce metrics on eng for every request received
// and every response sent.
func NewHandlerWith(eng *stats.Engine, h http.Handler) http.Handler {
	if eng == nil {
		eng = stats.DefaultEngine
	}
	return &handler{
		handler: h,
		eng:     eng,
	}
}

type handler struct {
	handler http.Handler
	eng     *stats.Engine
}

func (h *handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	req.Body = &requestBody{
		eng:  h.eng,
		req:  req,
		body: req.Body,
		op:   "read",
	}
	w := &responseWriter{
		ResponseWriter: res,
		eng:            h.eng,
		req:            req,
		start:          time.Now(),
	}
	defer w.complete()
	h.handler.ServeHTTP(w, req)
}

type responseWriter struct {
	http.ResponseWriter
	start       time.Time
	eng         *stats.Engine
	req         *http.Request
	status      int
	bytes       int
	wroteHeader bool
	wroteStats  bool
}

func (w *responseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *responseWriter) Write(b []byte) (n int, err error) {
	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = http.StatusOK
	}

	if n, err = w.ResponseWriter.Write(b); n > 0 {
		w.bytes += n
	}

	return
}

func (w *responseWriter) Hijack() (conn net.Conn, buf *bufio.ReadWriter, err error) {
	if conn, buf, err = w.ResponseWriter.(http.Hijacker).Hijack(); err == nil {
		w.wroteHeader = true
		w.complete()
	}
	return
}

func (w *responseWriter) complete() {
	if w.wroteStats {
		return
	}

	if !w.wroteHeader {
		w.wroteHeader = true
		w.status = http.StatusOK
	}

	now := time.Now()
	res := &http.Response{
		ProtoMajor:    w.req.ProtoMajor,
		ProtoMinor:    w.req.ProtoMinor,
		StatusCode:    w.status,
		Header:        w.Header(),
		Request:       w.req,
		ContentLength: -1,
	}

	m := metrics{w.eng}
	m.observeResponse(res, "write", w.bytes, now.Sub(w.start))

	w.req.Body.Close()
	w.wroteStats = true
}
