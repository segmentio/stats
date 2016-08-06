package stats

import (
	"io"
	"net/http"
	"path"
	"strconv"
)

func NewHttpHandler(client Client, handler http.Handler) http.Handler {
	return &httpHandler{
		handler:        handler,
		countReq:       client.Counter("http_request.count"),
		countRes:       client.Counter("http_response.count"),
		bytesReqHeader: client.Histogram("http_request_header.bytes"),
		bytesResHeader: client.Histogram("http_response_header.bytes"),
		bytesReqBody:   client.Histogram("http_request_body.bytes"),
		bytesResBody:   client.Histogram("http_response_body.bytes"),
		sizeReqHeader:  client.Histogram("http_request_header.size"),
		sizeResHeader:  client.Histogram("http_response_header.size"),
		timeReq:        client.Timer("http_request.duration"),
	}
}

type httpHandler struct {
	handler        http.Handler
	countReq       Counter
	countRes       Counter
	bytesReqHeader Histogram
	bytesResHeader Histogram
	bytesReqBody   Histogram
	bytesResBody   Histogram
	sizeReqHeader  Histogram
	sizeResHeader  Histogram
	timeReq        Timer
}

func (h *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tags := Tags{
		{"method", req.Method},
		{"path", path.Clean(req.URL.Path)},
		{"request_type", req.Header.Get("Content-Type")},
		{"request_encoding", req.Header.Get("Content-Encoding")},
	}

	c := &httpRequestContext{
		tags:    tags,
		handler: h,
		clock:   h.timeReq.Start(tags...),
	}

	r := &httpRequestReader{
		ReadCloser: req.Body,
	}

	w := &httpResponseWriter{
		ResponseWriter: res,
		clock:          c.clock,
	}

	req.Body = r
	h.countReq.Add(1, append(tags)...)
	h.handler.ServeHTTP(w, req)
	c.done(r, w, req)
}

type httpRequestContext struct {
	tags    Tags
	clock   Clock
	handler *httpHandler
}

func (c *httpRequestContext) done(r *httpRequestReader, w *httpResponseWriter, req *http.Request) {
	tags := append(c.tags, w.tags...)

	c.clock.Stamp("write_body", tags...)
	c.clock.Stop(tags...)

	bytesReqHeader := guessReqHeaderBytes(req)
	bytesResHeader := guessResHeaderBytes(req, w.status, w.Header())

	c.handler.bytesReqHeader.Observe(float64(bytesReqHeader), tags...)
	c.handler.bytesResHeader.Observe(float64(bytesResHeader), tags...)

	c.handler.bytesReqBody.Observe(float64(r.bytes), tags...)
	c.handler.bytesResBody.Observe(float64(w.bytes), tags...)

	c.handler.sizeReqHeader.Observe(float64(len(req.Header)), tags...)
	c.handler.sizeResHeader.Observe(float64(len(w.Header())), tags...)

	c.handler.countRes.Add(1, tags...)
}

type httpRequestReader struct {
	io.ReadCloser
	bytes int
}

func (r *httpRequestReader) Read(b []byte) (n int, err error) {
	if n, err = r.ReadCloser.Read(b); n > 0 {
		r.bytes += n
	}
	return
}

type httpResponseWriter struct {
	http.ResponseWriter
	bytes  int
	status int
	tags   Tags
	clock  Clock
}

func (w *httpResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)

		tagBucket := Tag{Name: "bucket"}
		tagStatus := Tag{Name: "status", Value: strconv.Itoa(status)}

		switch {
		case status < 100 || status >= 600:
			tagBucket.Value = "???"

		case status < 200:
			tagBucket.Value = "1xx"

		case status < 300:
			tagBucket.Value = "2xx"

		case status < 400:
			tagBucket.Value = "3xx"

		case status < 500:
			tagBucket.Value = "4xx"

		default:
			tagBucket.Value = "5xx"
		}

		h := w.Header()
		w.tags = append(w.tags,
			tagBucket,
			tagStatus,
			Tag{"response_type", h.Get("Content-Type")},
			Tag{"response_encoding", h.Get("Content-Encoding")},
		)

		w.clock.Stamp("write_header", w.tags...)
	}
}

func (w *httpResponseWriter) Write(b []byte) (n int, err error) {
	w.WriteHeader(http.StatusOK)

	if n, err = w.ResponseWriter.Write(b); n > 0 {
		w.bytes += n
	}

	return
}

func guessReqHeaderBytes(req *http.Request) (n int) {
	return len(req.Method) + len(req.URL.String()) + len(req.Proto) + 5 + guessHeaderBytes(req.Header)
}

func guessResHeaderBytes(req *http.Request, status int, hdr http.Header) (n int) {
	return len(req.Proto) + len(http.StatusText(status)) + 3 + guessHeaderBytes(hdr)
}

func guessHeaderBytes(h http.Header) (n int) {
	for k, v := range h {
		for _, s := range v {
			n += len(k) + len(s) + 4
		}
	}
	return
}
