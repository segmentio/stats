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

func init() {
	stats.Buckets.Set("http.message:header.size",
		5,
		10,
		20,
		40,
		80,
		math.Inf(+1),
	)

	stats.Buckets.Set("http.message:header.bytes",
		1e2, // 100 B
		1e3, // 1 KB
		1e4, // 10 KB
		1e5, // 100 KB
		1e6, // 1 MB
		math.Inf(+1),
	)

	stats.Buckets.Set("http.message:body.bytes",
		1e2, // 100 B
		1e3, // 1 KB
		1e4, // 10 KB
		1e5, // 100 KB
		1e6, // 1 MB
		1e7, // 10 MB
		1e8, // 100 MB
		1e9, // 1 GB
		math.Inf(+1),
	)

	stats.Buckets.Set("http:rtt.seconds",
		1*time.Millisecond,
		10*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
		10*time.Second,
		math.Inf(+1),
	)
}

type nullBody struct{}

func (n *nullBody) Close() error { return nil }

func (n *nullBody) Read(b []byte) (int, error) { return 0, io.EOF }

type requestBody struct {
	body    io.ReadCloser
	eng     *stats.Engine
	req     *http.Request
	metrics *metrics
	bytes   int
	op      string
	once    sync.Once
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
	r.metrics.observeRequest(r.req, r.op, r.bytes)
}

type responseBody struct {
	eng     *stats.Engine
	res     *http.Response
	metrics *metrics
	body    io.ReadCloser
	bytes   int
	op      string
	start   time.Time
	once    sync.Once
}

func (r *responseBody) Close() (err error) {
	err = r.body.Close()
	r.close()
	return
}

func (r *responseBody) Read(b []byte) (n int, err error) {
	if n, err = r.body.Read(b); n > 0 {
		r.bytes += n
	}
	return
}

func (r *responseBody) close() {
	r.once.Do(r.complete)
}

func (r *responseBody) complete() {
	r.metrics.observeResponse(r.res, r.op, r.bytes, time.Now().Sub(r.start))
	r.eng.ReportAt(r.start, r.metrics)
}

type metrics struct {
	http struct {
		err struct {
			count int `metric:"count" type:"counter"`
		} `metric:"error"`

		req struct {
			msg struct {
				count       int `metric:"count"        type:"counter"`
				headerSize  int `metric:"header.size"  type:"histogram"`
				headerBytes int `metric:"header.bytes" type:"histogram"`
				bodyBytes   int `metric:"body.bytes"   type:"histogram"`
			} `metric:"message"`

			operation string `tag:"operation"`
			msgtype   string `tag:"type"`
		}

		res struct {
			rtt time.Duration `metric:"rtt.seconds" type:"histogram"`

			msg struct {
				count       int `metric:"count"        type:"counter"`
				headerSize  int `metric:"header.size"  type:"histogram"`
				headerBytes int `metric:"header.bytes" type:"histogram"`
				bodyBytes   int `metric:"body.bytes"   type:"histogram"`
			} `metric:"message"`

			operation string `tag:"operation"`
			msgtype   string `tag:"type"`

			contentCharset   string `tag:"http_res_content_charset"`
			contentEncoding  string `tag:"http_res_content_endoing"`
			contentType      string `tag:"http_res_content_type"`
			protocol         string `tag:"http_res_protocol"`
			server           string `tag:"http_res_server"`
			statusBucket     string `tag:"http_res_status_bucket"`
			status           string `tag:"http_res_status"`
			transferEncoding string `tag:"http_res_transfer_encoding"`
			upgrade          string `tag:"http_res_upgrade"`
		}

		contentCharset   string `tag:"http_req_content_charset"`
		contentEncoding  string `tag:"http_req_content_endoing"`
		contentType      string `tag:"http_req_content_type"`
		host             string `tag:"http_req_host"`
		method           string `tag:"http_req_method"`
		protocol         string `tag:"http_req_protocol"`
		transferEncoding string `tag:"http_req_transfer_encoding"`
	} `metric:"http"`
}

func (m *metrics) observeRequest(req *http.Request, op string, bodyLen int) {
	contentType, charset := contentType(req.Header)
	contentEncoding := contentEncoding(req.Header)
	transferEncoding := transferEncoding(req.TransferEncoding)
	host := requestHost(req)

	m.http.req.msg.count = 1
	m.http.req.msg.headerSize = len(req.Header)
	m.http.req.msg.headerBytes = requestHeaderLength(req)
	m.http.req.msg.bodyBytes = bodyLen

	m.http.req.operation = op
	m.http.req.msgtype = "request"

	m.http.contentCharset = charset
	m.http.contentEncoding = contentEncoding
	m.http.contentType = contentType
	m.http.host = host
	m.http.method = req.Method
	m.http.protocol = req.Proto
	m.http.transferEncoding = transferEncoding
}

func (m *metrics) observeResponse(res *http.Response, op string, bodyLen int, rtt time.Duration) {
	contentType, charset := contentType(res.Header)
	contentEncoding := contentEncoding(res.Header)
	upgrade := headerValue(res.Header, "Upgrade")
	server := headerValue(res.Header, "Server")
	bucket := responseStatusBucket(res.StatusCode)
	status := statusCode(res.StatusCode)
	transferEncoding := transferEncoding(res.TransferEncoding)

	m.http.res.msg.count = 1
	m.http.res.msg.headerSize = len(res.Header)
	m.http.res.msg.headerBytes = responseHeaderLength(res)
	m.http.res.msg.bodyBytes = bodyLen
	m.http.res.rtt = rtt

	m.http.res.operation = op
	m.http.res.msgtype = "response"

	m.http.res.contentCharset = charset
	m.http.res.contentEncoding = contentEncoding
	m.http.res.contentType = contentType
	m.http.res.protocol = res.Proto
	m.http.res.server = server
	m.http.res.statusBucket = bucket
	m.http.res.status = status
	m.http.res.transferEncoding = transferEncoding
	m.http.res.upgrade = upgrade
}

func (m *metrics) observeError(rtt time.Duration) {
	m.http.err.count = 1
	m.http.res.rtt = rtt
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
		if host = headerValue(req.Header, "Host"); len(host) == 0 {
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
	return parseContentType(headerValue(h, "Content-Type"))
}

func contentEncoding(h http.Header) string {
	return strings.TrimSpace(headerValue(h, "Content-Encoding"))
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

// headerValues is equivalent to http.Header.Get but assumes that the keys and
// the header name to lookup are already in their canonical form so we can save
// the expansive call to net/textproto.CanonicalMIMEHeaderKey.
func headerValue(header http.Header, name string) string {
	values := header[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// statusCode behaves like strconv.Itoa but uses a lookup table to avoid having
// to do a dynamic memory allocation to convert common http status codes to a
// string representation.
func statusCode(code int) string {
	if code >= 100 {
		switch {
		case code < 200:
			return statusCodeWithTable(code-100, statusCode100[:])
		case code < 300:
			return statusCodeWithTable(code-200, statusCode200[:])
		case code < 400:
			return statusCodeWithTable(code-300, statusCode300[:])
		case code < 500:
			return statusCodeWithTable(code-400, statusCode400[:])
		case code < 600:
			return statusCodeWithTable(code-500, statusCode500[:])
		}
	}
	return strconv.Itoa(code)
}

func statusCodeWithTable(code int, table []string) string {
	if code < len(table) {
		return table[code]
	}
	return strconv.Itoa(code)
}

var statusCode100 = [...]string{
	"100", "101",
}

var statusCode200 = [...]string{
	"200", "201", "202", "203", "204", "205", "206",
}

var statusCode300 = [...]string{
	"300", "301", "302", "303", "304", "305", "306", "307",
}

var statusCode400 = [...]string{
	"400", "401", "402", "403", "404", "405", "406", "407", "408", "409",
	"410", "411", "412", "413", "414", "415", "416", "417", "418", "419",
	"420", "421", "422", "423", "424", "425", "426", "427", "428", "429",
}

var statusCode500 = [...]string{
	"500", "501", "502", "503", "504", "505",
}
