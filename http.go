package stats

import (
	"bufio"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
)

func NewHttpTransport(client Client, roundTripper http.RoundTripper) http.RoundTripper {
	return httpTransport{
		roundTripper: roundTripper,
		stats:        newHttpStats(client, "http_client"),
	}
}

func NewHttpHandler(client Client, handler http.Handler) http.Handler {
	return httpHandler{
		handler: handler,
		stats:   newHttpStats(client, "http_server"),
	}
}

type httpTransport struct {
	roundTripper http.RoundTripper
	stats        *httpStats
}

func (t httpTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	tags := httpRequestTags(req)
	clock := t.stats.timeReq.Start()

	r := &httpStatsReport{
		req:  req,
		tags: tags,
	}

	t.setup(r, req, clock)

	res, err = t.roundTripper.RoundTrip(req)
	req.Body.Close() // safe guard, the roundtripper should have done it already

	if err != nil {
		t.teardownWithError(r, err, clock)
	} else {
		t.teardownWithResponse(r, res, clock)
	}

	return
}

func (t httpTransport) setup(report *httpStatsReport, req *http.Request, clock Clock) {
	r := &CountReader{R: req.Body}
	c := io.Closer(req.Body)

	once1 := &sync.Once{}
	once2 := &sync.Once{}

	do1 := func() { clock.Stamp("write_header", report.tags...) }
	do2 := func() { clock.Stamp("write_body", report.tags...) }

	req.Body = readCloser{
		Reader: readerFunc(func(b []byte) (int, error) {
			once1.Do(do1)
			return r.Read(b)
		}),
		Closer: closerFunc(func() (err error) {
			err = c.Close()
			once1.Do(do1)
			once2.Do(do2)
			report.reqBodyBytes = r.N
			return
		}),
	}

	return
}

func (t httpTransport) teardownWithError(report *httpStatsReport, err error, clock Clock) {
	report.err = err
	clock.Stop(report.tags...)
	t.stats.report(*report)
}

func (t httpTransport) teardownWithResponse(report *httpStatsReport, res *http.Response, clock Clock) {
	report.res = res
	report.tags = append(report.tags, httpResponseTags(res.StatusCode, res.Header)...)
	clock.Stamp("read_header", report.tags...)

	r := &CountReader{R: res.Body}
	c := io.Closer(res.Body)

	once1 := &sync.Once{}
	once2 := &sync.Once{}

	do := func() { clock.Stamp("read_body", report.tags...) }

	res.Body = readCloser{
		Reader: readerFunc(func(b []byte) (n int, err error) {
			if n, err = r.Read(b); err == io.EOF {
				once1.Do(do)
			}
			return
		}),
		Closer: closerFunc(func() (err error) {
			err = c.Close()
			once1.Do(do)
			once2.Do(func() {
				clock.Stop(report.tags...)
				report.err = err
				report.resBodyBytes = r.N
				t.stats.report(*report)
			})
			return
		}),
	}
}

type httpHandler struct {
	handler http.Handler
	stats   *httpStats
}

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tags := httpRequestTags(req)
	clock := h.stats.timeReq.Start()
	clock.Stamp("read_header", tags...)

	c := io.Closer(req.Body)

	r := &CountReader{
		R: req.Body,
	}

	w := &httpResponseWriter{
		ResponseWriter: res,
		tags:           tags,
		clock:          clock,
	}

	once := &sync.Once{}
	body := func() { clock.Stamp("read_body", tags...) }

	req.Body = readCloser{
		Reader: readerFunc(func(b []byte) (n int, err error) {
			if n, err = r.Read(b); err == io.EOF {
				once.Do(body)
			}
			return
		}),
		Closer: closerFunc(func() (err error) {
			err = c.Close()
			once.Do(body)
			return
		}),
	}

	res = w

	if _, ok := w.ResponseWriter.(http.Hijacker); ok {
		res = httpResponseHijacker{w}
	}

	h.stats.countReq.Add(1, tags...)
	h.handler.ServeHTTP(res, req)
	req.Body.Close()

	clock.Stamp("write_body", w.tags...)
	clock.Stop(w.tags...)

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
	})
}

type httpStats struct {
	countReq       Counter
	countRes       Counter
	countErr       Counter
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
		countErr:       client.Counter(namespace + ".error.count"),
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
	err          error
	reqBodyBytes int
	resBodyBytes int
	tags         Tags
}

func (s *httpStats) report(r httpStatsReport) {
	if r.err != nil {
		s.countErr.Add(1, r.tags...)
	}

	if r.req != nil {
		reqHeaderBytes := httpRequestHeaderLength(r.req)
		s.bytesReqHeader.Observe(float64(reqHeaderBytes), r.tags...)
		s.bytesReqBody.Observe(float64(r.reqBodyBytes), r.tags...)
		s.sizeReqHeader.Observe(float64(len(r.req.Header)), r.tags...)
	}

	if r.res != nil {
		resHeaderBytes := httpResponseHeaderLength(r.res)
		s.bytesResHeader.Observe(float64(resHeaderBytes), r.tags...)
		s.bytesResBody.Observe(float64(r.resBodyBytes), r.tags...)
		s.sizeResHeader.Observe(float64(len(r.res.Header)), r.tags...)
		s.countRes.Add(1, r.tags...)
	}
}

type httpResponseHijacker struct {
	*httpResponseWriter
}

func (w httpResponseHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
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
		w.tags = append(w.tags, httpResponseTags(status, w.Header())...)
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

func httpResponseTags(status int, header http.Header) Tags {
	return Tags{
		{"bucket", httpResponseStatusBucket(status)},
		{"status", strconv.Itoa(status)},
		{"response_type", header.Get("Content-Type")},
		{"response_encoding", header.Get("Content-Encoding")},
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
