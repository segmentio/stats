package httpstats

import (
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"path"
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

func MakeClientMetrics(eng *stats.Engine, tags ...stats.Tag) Metrics {
	return makeMetrics(eng, "write", "read", tags...)
}

func MakeServerMetrics(eng *stats.Engine, tags ...stats.Tag) Metrics {
	return makeMetrics(eng, "read", "write", tags...)
}

func makeMetrics(eng *stats.Engine, reqOp string, resOp string, tags ...stats.Tag) Metrics {
	return Metrics{
		Requests:  makeMessageMetrics(eng, "request", reqOp, tags...),
		Responses: makeMessageMetrics(eng, "response", resOp, tags...),
		Errors:    stats.MakeCounter(eng, "http.errors.count", tags...),
		RTT:       stats.MakeTimer(eng, "http.rtt.seconds", tags...),
	}
}

func makeMessageMetrics(eng *stats.Engine, typ string, op string, tags ...stats.Tag) MessageMetrics {
	tags = append(tags, stats.Tag{"type", typ}, stats.Tag{"operation", op})
	return MessageMetrics{
		Count:       stats.MakeCounter(eng, "http.message.count", tags...),
		HeaderSizes: stats.MakeHistogram(eng, "http.message.header.sizes", tags...),
		HeaderBytes: stats.MakeHistogram(eng, "http.message.header.bytes", tags...),
		BodyBytes:   stats.MakeHistogram(eng, "http.message.body.bytes", tags...),
	}
}

func (m *Metrics) ObserveRequest(req *http.Request) (body io.ReadCloser, tags []stats.Tag) {
	tags = makeRequestTags(req)

	m.Requests.Count.Clone(tags...).Incr()
	m.Requests.HeaderSizes.Clone(tags...).Observe(float64(len(req.Header)))
	m.Requests.HeaderBytes.Clone(tags...).Observe(float64(requestHeaderLength(req)))

	body = m.makeMessageBody(req.Body, nil)
	return
}

func (m *Metrics) ObserveResponse(res *http.Response, clock *stats.Clock) (body io.ReadCloser, tags []stats.Tag) {
	tags = makeResponseTags(res)

	m.Responses.Count.Clone(tags...).Incr()
	m.Responses.HeaderSizes.Clone(tags...).Observe(float64(len(res.Header)))
	m.Responses.HeaderBytes.Clone(tags...).Observe(float64(responseHeaderLength(res)))

	body = m.makeMessageBody(res.Body, clock, tags...)
	return
}

func (m *Metrics) makeMessageBody(body io.ReadCloser, clock *stats.Clock, tags ...stats.Tag) io.ReadCloser {
	once := &sync.Once{}
	read := &iostats.CountReader{R: body}
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: read,
		Closer: iostats.CloserFunc(func() (err error) {
			if body != nil {
				err = body.Close()
			}
			once.Do(func() {
				m.Responses.BodyBytes.Clone(tags...).Observe(float64(read.N))
				if clock != nil {
					clock.Clone(tags...).Stop()
				}
			})
			return
		}),
	}
}

func makeRequestTags(req *http.Request) []stats.Tag {
	host, _ := requestHost(req)
	ctype, charset := contentType(req.Header)
	return []stats.Tag{
		{"http_req_method", req.Method},
		{"http_req_path", sanitizeHttpPath(req.URL.Path)},
		{"http_req_protocol", req.Proto},
		{"http_req_host", host},
		{"http_req_content_type", ctype},
		{"http_req_content_charset", charset},
		{"http_req_content_encoding", contentEncoding(req.Header)},
		{"http_req_transfer_encoding", transferEncoding(req.TransferEncoding)},
	}
}

func makeResponseTags(res *http.Response) []stats.Tag {
	ctype, charset := contentType(res.Header)
	tags := []stats.Tag{
		{"http_res_status_bucket", responseStatusBucket(res.StatusCode)},
		{"http_res_status", strconv.Itoa(res.StatusCode)},
		{"http_res_protocol", res.Proto},
		{"http_res_server", res.Header.Get("Server")},
		{"http_res_upgrade", res.Header.Get("Upgrade")},
		{"http_res_content_type", ctype},
		{"http_res_content_charset", charset},
		{"http_res_content_encoding", contentEncoding(res.Header)},
		{"http_res_transfer_encoding", transferEncoding(res.TransferEncoding)},
	}

	if req := res.Request; req != nil {
		tags = append(tags, makeRequestTags(req)...)
	}

	return tags
}

func requestHeaderLength(req *http.Request) int {
	n := headerLength(req.Header) +
		urlLength(req.URL) +
		len(" ") +
		len(req.Method) +
		len(" ") +
		len(req.Proto) +
		len("\r\n")

	if _, ok := req.Header["User-Agent"]; !ok {
		n += len("User-Agent: Go-http-client/1.1\r\n")
	}

	n += len("Host: ") + len(req.Host) + len("\r\n")
	return n
}

func responseHeaderLength(res *http.Response) int {
	n := headerLength(res.Header) +
		len(res.Proto) +
		len(" ") +
		intLength(int64(res.StatusCode)) +
		len(" ") +
		len(http.StatusText(res.StatusCode)) +
		len("\r\n")

	if _, ok := res.Header["Connection"]; !ok {
		n += len("Connection: close\r\n")
	}

	return n
}

func headerLength(h http.Header) int {
	n := 0

	for name, values := range h {
		for _, v := range values {
			n += len(name) + len(": ") + len(v) + len("\r\n")
		}
	}

	return n + len("\r\n")
}

func urlLength(u *url.URL) int {
	n := len(u.Host) + len(u.Path)

	if l := len(u.Scheme); l != 0 {
		n += l + len("://")
	}

	if l := len(u.RawQuery); l != 0 {
		n += l + len("?")
	}

	if user := u.User; user != nil {
		n += len(user.Username()) + len("@")

		if p, ok := user.Password(); ok {
			n += len(p) + len(":")
		}
	}

	if len(u.Fragment) != 0 {
		n += len(u.Fragment) + len("#")
	}

	return n
}

func intLength(n int64) int {
	return int(math.Log10(float64(n))) + 1
}

func requestHost(req *http.Request) (host string, port string) {
	if host = req.Host; len(host) == 0 {
		if host = req.URL.Host; len(host) == 0 {
			host = req.Header.Get("Host")
		}
	}

	if h, p, e := net.SplitHostPort(host); e == nil {
		host, port = h, p
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

func contentType(h http.Header) (string, string) {
	return parseContentType(h.Get("Content-Type"))
}

func contentEncoding(h http.Header) string {
	return strings.TrimSpace(h.Get("Content-Encoding"))
}

func transferEncoding(te []string) string {
	switch len(te) {
	case 0:
		return ""
	case 1:
		return te[0]
	default:
		return strings.Join(te, ";")
	}
}

func parseContentType(s string) (contentType string, charset string) {
	for i := 0; len(s) != 0; i++ {
		var t string
		if t, s = parseHeaderToken(s); strings.HasPrefix(t, "charset=") {
			charset = t[8:]
		} else if len(contentType) == 0 {
			contentType = t
		}
	}
	return
}

func parseHeaderToken(s string) (token string, next string) {
	if i := strings.IndexByte(s, ';'); i >= 0 {
		token, next = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	} else {
		token = strings.TrimSpace(s)
	}
	return
}

func isIDByte(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-'
}

func isID(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := range s {
		if !isIDByte(s[i]) {
			return false
		}
	}
	return true
}

func sanitizeHttpPath(p string) string {
	if len(p) == 0 {
		return p
	}
	parts := strings.Split(path.Clean(p), "/")
	for i, s := range parts {
		if isID(s) {
			parts[i] = "<id>"
		}
	}
	return strings.Join(parts, "/")
}

type readCloser struct {
	io.Reader
	io.Closer
}

type nopeReadCloser struct{}

func (n nopeReadCloser) Close() error { return nil }

func (n nopeReadCloser) Read(b []byte) (int, error) { return 0, io.EOF }
