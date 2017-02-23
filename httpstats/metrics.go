package httpstats

import (
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

type nullBody struct{}

func (n *nullBody) Close() error { return nil }

func (n *nullBody) Read(b []byte) (int, error) { return 0, io.EOF }

type requestBody struct {
	body  io.ReadCloser
	eng   *stats.Engine
	req   *http.Request
	bytes int
	op    string
	once  sync.Once
}

func (r *requestBody) Close() (err error) {
	err = r.body.Close()
	r.close()
	return
}

func (r *requestBody) Read(b []byte) (n int, err error) {
	if n, err = r.body.Read(b); n > 0 {
		r.bytes += n
	}
	return
}

func (r *requestBody) close() {
	r.once.Do(r.complete)
}

func (r *requestBody) complete() {
	m := metrics{r.eng}
	m.observeRequest(r.req, r.op, r.bytes)
}

type responseBody struct {
	eng   *stats.Engine
	res   *http.Response
	body  io.ReadCloser
	bytes int
	op    string
	start time.Time
	once  sync.Once
}

func (r *responseBody) Close() (err error) {
	err = r.body.Close()
	r.once.Do(r.complete)
	return
}

func (r *responseBody) Read(b []byte) (n int, err error) {
	if n, err = r.body.Read(b); n > 0 {
		r.bytes += n
	}
	return
}

func (r *responseBody) complete() {
	m := metrics{r.eng}
	m.observeResponse(r.res, r.op, r.bytes, time.Now().Sub(r.start))
}

type metrics struct {
	eng *stats.Engine
}

func (m metrics) incrMessageCount(tags ...stats.Tag) {
	m.eng.Incr("http.message.count", tags...)
}

func (m metrics) incrErrorCount(tags ...stats.Tag) {
	m.eng.Incr("http.error.count", tags...)
}

func (m metrics) observeHeaderSize(size int, tags ...stats.Tag) {
	m.eng.Observe("http.message.header.size", float64(size), tags...)
}

func (m metrics) observeHeaderLength(len int, tags ...stats.Tag) {
	m.eng.Observe("http.message.header.bytes", float64(len), tags...)
}

func (m metrics) observeBodyLength(len int, tags ...stats.Tag) {
	m.eng.Observe("http.message.body.bytes", float64(len), tags...)
}

func (m metrics) observeRTT(rtt time.Duration, tags ...stats.Tag) {
	m.eng.Observe("http.rtt.seconds", rtt.Seconds(), tags...)
}

func (m metrics) observeRequest(req *http.Request, op string, bodyLen int) {
	var a [10]stats.Tag
	var t = a[:0]

	t = append(t, stats.Tag{"type", "request"})
	t = append(t, stats.Tag{"operation", op})
	t = appendRequestTags(t, req)

	m.incrMessageCount(t...)
	m.observeHeaderSize(len(req.Header), t...)
	m.observeHeaderLength(requestHeaderLength(req), t...)
	m.observeBodyLength(bodyLen, t...)
}

func (m metrics) observeResponse(res *http.Response, op string, bodyLen int, rtt time.Duration) {
	var a [20]stats.Tag
	var t = a[:0]

	t = append(t, stats.Tag{"type", "response"})
	t = append(t, stats.Tag{"operation", op})
	t = appendResponseTags(t, res)

	if req := res.Request; req != nil {
		t = appendRequestTags(t, req)
	}

	m.incrMessageCount(t...)
	m.observeHeaderSize(len(res.Header), t...)
	m.observeHeaderLength(responseHeaderLength(res), t...)
	m.observeBodyLength(bodyLen, t...)
	m.observeRTT(rtt, t...)
}

func (m metrics) observeError(req *http.Request, op string) {
	var a [10]stats.Tag
	var t = a[:0]

	t = append(t, stats.Tag{"type", "request"})
	t = append(t, stats.Tag{"operation", op})
	t = appendRequestTags(t, req)

	m.incrErrorCount(t...)
}

func appendRequestTags(tags []stats.Tag, req *http.Request) []stats.Tag {
	ctype, charset := contentType(req.Header)
	return append(tags,
		stats.Tag{"http_req_content_charset", charset},
		stats.Tag{"http_req_content_encoding", contentEncoding(req.Header)},
		stats.Tag{"http_req_content_type", ctype},
		stats.Tag{"http_req_host", requestHost(req)},
		stats.Tag{"http_req_method", req.Method},
		stats.Tag{"http_req_path", req.URL.Path},
		stats.Tag{"http_req_protocol", req.Proto},
		stats.Tag{"http_req_transfer_encoding", transferEncoding(req.TransferEncoding)},
	)
}

func appendResponseTags(tags []stats.Tag, res *http.Response) []stats.Tag {
	ctype, charset := contentType(res.Header)
	return append(tags,
		stats.Tag{"http_res_content_charset", charset},
		stats.Tag{"http_res_content_encoding", contentEncoding(res.Header)},
		stats.Tag{"http_res_content_type", ctype},
		stats.Tag{"http_res_protocol", res.Proto},
		stats.Tag{"http_res_server", res.Header.Get("Server")},
		stats.Tag{"http_res_status_bucket", responseStatusBucket(res.StatusCode)},
		stats.Tag{"http_res_status", strconv.Itoa(res.StatusCode)},
		stats.Tag{"http_res_transfer_encoding", transferEncoding(res.TransferEncoding)},
		stats.Tag{"http_res_upgrade", res.Header.Get("Upgrade")},
	)
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

func requestHost(req *http.Request) (host string) {
	if host = req.Host; len(host) == 0 {
		if host = req.Header.Get("Host"); len(host) == 0 {
			host = req.URL.Host
		}
	}
	return
}

func responseStatusBucket(status int) string {
	switch {
	case status < 100 || status >= 600:
		return ""

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

func contentType(h http.Header) (string, string) {
	return parseContentType(h.Get("Content-Type"))
}

func contentEncoding(h http.Header) string {
	return strings.TrimSpace(h.Get("Content-Encoding"))
}

func transferEncoding(te []string) string {
	switch len(te) {
	case 0:
		return "identity"
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
