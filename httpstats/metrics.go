package httpstats

import (
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/iostats"
)

type Metrics struct {
	Requests  MessageMetrics
	Responses MessageMetrics
	Errors    stats.Counter
	RTT       stats.Timer
}

type MessageMetrics struct {
	Count       stats.Counter
	HeaderSizes stats.Histogram
	HeaderBytes stats.Histogram
	BodyBytes   stats.Histogram
}

func NewClientMetrics(client stats.Client, tags ...stats.Tag) *Metrics {
	return newMetrics(client, "write", "read", tags...)
}

func NewServerMetrics(client stats.Client, tags ...stats.Tag) *Metrics {
	return newMetrics(client, "read", "write", tags...)
}

func newMetrics(client stats.Client, reqOp string, resOp string, tags ...stats.Tag) *Metrics {
	return &Metrics{
		Requests:  makeMessageMetrics(client, "request", reqOp, tags...),
		Responses: makeMessageMetrics(client, "response", resOp, tags...),
		Errors:    client.Counter("http.errors.count", tags...),
		RTT:       client.Timer("http.rtt.seconds", tags...),
	}
}

func makeMessageMetrics(client stats.Client, typ string, op string, tags ...stats.Tag) MessageMetrics {
	tags = append(tags, stats.Tag{"type", typ}, stats.Tag{"operation", op})
	return MessageMetrics{
		Count:       client.Counter("http.message.count", tags...),
		HeaderSizes: client.Histogram("http.message.header.sizes", tags...),
		HeaderBytes: client.Histogram("http.message.header.bytes", tags...),
		BodyBytes:   client.Histogram("http.message.body.bytes", tags...),
	}
}

func (m *Metrics) ObserveRequest(req *http.Request, tags ...stats.Tag) (time stats.Clock, body io.ReadCloser) {
	time = m.RTT.Start()
	tags = makeRequestTags(req, tags...)

	m.Requests.Count.Add(1, tags...)
	m.Requests.HeaderSizes.Observe(float64(len(req.Header)), tags...)
	m.Requests.HeaderBytes.Observe(float64(requestHeaderLength(req)), tags...)

	body = m.makeMessageBody(req.Body, nil, tags...)
	return
}

func (m *Metrics) ObserveResponse(res *http.Response, time stats.Clock, tags ...stats.Tag) (body io.ReadCloser) {
	tags = makeResponseTags(res, tags...)

	m.Responses.Count.Add(1, tags...)
	m.Responses.HeaderSizes.Observe(float64(len(res.Header)), tags...)
	m.Responses.HeaderBytes.Observe(float64(responseHeaderLength(res)), tags...)

	body = m.makeMessageBody(res.Body, time, tags...)
	return
}

func (m *Metrics) makeMessageBody(body io.ReadCloser, time stats.Clock, tags ...stats.Tag) io.ReadCloser {
	once := &sync.Once{}
	read := &iostats.CountReader{R: body}
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: read,
		Closer: iostats.CloserFunc(func() (err error) {
			err = body.Close()
			once.Do(func() {
				m.Responses.BodyBytes.Observe(float64(read.N), tags...)
				if time != nil {
					time.Stop(tags...)
				}
			})
			return
		}),
	}
}

func makeRequestTags(req *http.Request, tags ...stats.Tag) stats.Tags {
	return append(stats.Tags{
		{"http_req_method", req.Method},
		{"http_req_path", req.URL.Path},
		{"http_req_protocol", req.Proto},
		{"http_req_host", requestHost(req)},
		{"http_req_content_type", req.Header.Get("Content-Type")},
		{"http_req_content_encoding", req.Header.Get("Content-Encoding")},
		{"http_req_transfer_encoding", strings.Join(req.TransferEncoding, ",")},
	}, tags...)
}

func makeResponseTags(res *http.Response, tags ...stats.Tag) stats.Tags {
	tags = append(stats.Tags{
		{"http_res_status_bucket", responseStatusBucket(res.StatusCode)},
		{"http_res_status", strconv.Itoa(res.StatusCode)},
		{"http_res_protocol", res.Proto},
		{"http_res_server", res.Header.Get("Server")},
		{"http_res_upgrade", res.Header.Get("Upgrade")},
		{"http_res_content_type", res.Header.Get("Content-Type")},
		{"http_res_content_encoding", res.Header.Get("Content-Encoding")},
		{"http_res_transfer_encoding", strings.Join(res.TransferEncoding, ",")},
	}, tags...)

	if req := res.Request; req != nil {
		tags = append(tags, makeRequestTags(req)...)
	}

	return tags
}

func requestHeaderLength(req *http.Request) int {
	w := &iostats.CountWriter{W: ioutil.Discard}
	r := &http.Request{
		Method:           req.Method,
		URL:              req.URL,
		Host:             req.Host,
		ContentLength:    -1,
		TransferEncoding: req.TransferEncoding,
		Header:           copyHeader(req.Header),
		Body:             nopeReadCloser{},
	}

	if req.ContentLength >= 0 {
		r.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	}

	r.Write(w)
	return w.N
}

func responseHeaderLength(res *http.Response) int {
	w := &iostats.CountWriter{W: ioutil.Discard}
	r := &http.Response{
		StatusCode:       res.StatusCode,
		ProtoMajor:       res.ProtoMajor,
		ProtoMinor:       res.ProtoMinor,
		Request:          res.Request,
		TransferEncoding: res.TransferEncoding,
		Trailer:          res.Trailer,
		ContentLength:    -1,
		Header:           copyHeader(res.Header),
		Body:             nopeReadCloser{},
	}

	if res.ContentLength >= 0 {
		r.Header.Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
	}

	r.Write(w)
	return w.N
}

func requestHost(req *http.Request) (host string) {
	if host = req.Host; len(host) == 0 {
		if host = req.URL.Host; len(host) == 0 {
			host = req.Header.Get("Host")
		}
	}
	return
}

func responseStatusBucket(status int) string {
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

func copyHeader(h http.Header) http.Header {
	copy := make(http.Header, len(h))
	for name, value := range h {
		copy[name] = value
	}
	return copy
}

type readCloser struct {
	io.Reader
	io.Closer
}

type nopeReadCloser struct{}

func (n nopeReadCloser) Close() error { return nil }

func (n nopeReadCloser) Read(b []byte) (int, error) { return 0, io.EOF }
