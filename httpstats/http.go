package httpstats

import (
	"bufio"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/iostats"
)

type httpStats struct {
	countReq       stats.Counter
	countRes       stats.Counter
	countErr       stats.Counter
	bytesReqHeader stats.Histogram
	bytesResHeader stats.Histogram
	bytesReqBody   stats.Histogram
	bytesResBody   stats.Histogram
	sizeReqHeader  stats.Histogram
	sizeResHeader  stats.Histogram
	timeReq        stats.Timer
}

func newHttpStats(client stats.Client, namespace string) *httpStats {
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
	tags         stats.Tags
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
	tags   stats.Tags
	clock  stats.Clock
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

func httpRequestTags(req *http.Request) stats.Tags {
	return stats.Tags{
		{"method", req.Method},
		{"path", req.URL.Path},
		{"request_type", req.Header.Get("Content-Type")},
		{"request_encoding", req.Header.Get("Content-Encoding")},
	}
}

func httpResponseTags(status int, header http.Header) stats.Tags {
	return stats.Tags{
		{"bucket", httpResponseStatusBucket(status)},
		{"status", strconv.Itoa(status)},
		{"response_type", header.Get("Content-Type")},
		{"response_encoding", header.Get("Content-Encoding")},
	}
}

func httpRequestHeaderLength(req *http.Request) int {
	w := &iostats.CountWriter{W: ioutil.Discard}
	r := &http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Host:             req.Host,
		ContentLength:    -1,
		TransferEncoding: req.TransferEncoding,
		Header:           copyHttpHeader(req.Header),
		Body:             iostats.NopeReadCloser{},
	}

	if req.ContentLength >= 0 {
		r.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	}

	r.Write(w)
	return w.N
}

func httpResponseHeaderLength(res *http.Response) int {
	w := &iostats.CountWriter{W: ioutil.Discard}
	r := &http.Response{
		StatusCode:       res.StatusCode,
		ProtoMajor:       res.ProtoMajor,
		ProtoMinor:       res.ProtoMinor,
		Request:          res.Request,
		TransferEncoding: res.TransferEncoding,
		Trailer:          res.Trailer,
		ContentLength:    -1,
		Header:           copyHttpHeader(res.Header),
		Body:             iostats.NopeReadCloser{},
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
