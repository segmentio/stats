package stats

import (
	"io/ioutil"
	"net/http"
	"strconv"
)

func NewHttpHandler(client Client, handler http.Handler) http.Handler {
	return httpHandler{
		handler: handler,
		stats:   newHttpStats(client, "http_server"),
	}
}

type httpHandler struct {
	handler http.Handler
	stats   *httpStats
}

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tags := httpRequestTags(req)
	clock := h.stats.timeReq.Start(tags...)

	r := &CountReader{
		R: req.Body,
	}

	w := &httpResponseWriter{
		ResponseWriter: res,
		tags:           tags,
		clock:          clock,
	}

	req.Body = readCloser{r, req.Body}

	h.stats.countReq.Add(1, tags...)
	h.handler.ServeHTTP(w, req)
	h.stats.report(httpStatsReport{
		req: req,
		res: &http.Response{
			StatusCode:    w.status,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			ContentLength: -1,
			Request:       req,
			Header:        w.Header(),
			Body:          nopeReadCloser{},
		},
		reqBodyBytes: r.N,
		resBodyBytes: w.bytes,
		tags:         w.tags,
		clock:        clock,
	})
}

type httpStats struct {
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

func newHttpStats(client Client, namespace string) *httpStats {
	return &httpStats{
		countReq:       client.Counter(namespace + ".request.count"),
		countRes:       client.Counter(namespace + ".response.count"),
		bytesReqHeader: client.Histogram(namespace + ".request_header.bytes"),
		bytesResHeader: client.Histogram(namespace + ".response_header.bytes"),
		bytesReqBody:   client.Histogram(namespace + ".request_body.bytes"),
		bytesResBody:   client.Histogram(namespace + ".response_body.bytes"),
		sizeReqHeader:  client.Histogram(namespace + ".request_header.size"),
		sizeResHeader:  client.Histogram(namespace + ".response_header.size"),
		timeReq:        client.Timer(namespace + ".request.duration"),
	}
}

type httpStatsReport struct {
	req          *http.Request
	res          *http.Response
	reqBodyBytes int
	resBodyBytes int
	tags         Tags
	clock        Clock
}

func (s *httpStats) report(r httpStatsReport) {
	r.clock.Stamp("write_body", r.tags...)
	r.clock.Stop(r.tags...)

	reqHeaderBytes := httpRequestHeaderLength(r.req)
	resHeaderBytes := httpResponseHeaderLength(r.res)

	s.bytesReqHeader.Observe(float64(reqHeaderBytes), r.tags...)
	s.bytesResHeader.Observe(float64(resHeaderBytes), r.tags...)

	s.bytesReqBody.Observe(float64(r.reqBodyBytes), r.tags...)
	s.bytesResBody.Observe(float64(r.resBodyBytes), r.tags...)

	s.sizeReqHeader.Observe(float64(len(r.req.Header)), r.tags...)
	s.sizeResHeader.Observe(float64(len(r.res.Header)), r.tags...)

	s.countRes.Add(1, r.tags...)
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

		tagBucket := Tag{Name: "bucket", Value: httpResponseStatusBucket(status)}
		tagStatus := Tag{Name: "status", Value: strconv.Itoa(status)}

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

func httpRequestTags(req *http.Request) Tags {
	return Tags{
		{"method", req.Method},
		{"path", req.URL.Path},
		{"request_type", req.Header.Get("Content-Type")},
		{"request_encoding", req.Header.Get("Content-Encoding")},
	}
}

func httpRequestHeaderLength(req *http.Request) int {
	w := &CountWriter{W: ioutil.Discard}
	r := &http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Host:             req.Host,
		ContentLength:    -1,
		TransferEncoding: req.TransferEncoding,
		Header:           copyHttpHeader(req.Header),
		Body:             nopeReadCloser{},
	}

	if req.ContentLength >= 0 {
		r.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	}

	r.Write(w)
	return w.N
}

func httpResponseHeaderLength(res *http.Response) int {
	w := &CountWriter{W: ioutil.Discard}
	r := &http.Response{
		StatusCode:       res.StatusCode,
		ProtoMajor:       res.ProtoMajor,
		ProtoMinor:       res.ProtoMinor,
		Request:          res.Request,
		TransferEncoding: res.TransferEncoding,
		Trailer:          res.Trailer,
		ContentLength:    -1,
		Header:           copyHttpHeader(res.Header),
		Body:             nopeReadCloser{},
	}

	if res.ContentLength >= 0 {
		r.Header.Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
	}

	r.Write(w)
	return w.N
}

func httpResponseStatusBucket(status int) string {
	switch {
	case status < 100 || status >= 600:
		return "???"

	case status < 200:
		return "1xx"

	case status < 300:
		return "2xx"

	case status < 400:
		return "3xx"

	case status < 500:
		return "4xx"

	default:
		return "5xx"
	}
}

func copyHttpHeader(hdr http.Header) http.Header {
	copy := make(http.Header, len(hdr))

	for name, value := range hdr {
		copy[name] = value
	}

	return copy
}
