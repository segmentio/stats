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
	"github.com/segmentio/stats/iostats"
)

type Metrics struct {
	eng     *stats.Engine
	reqTags []stats.Tag
	resTags []stats.Tag
}

func MakeClientMetrics(eng *stats.Engine, tags ...stats.Tag) Metrics {
	return makeMetrics(eng, "write", "read", tags...)
}

func MakeServerMetrics(eng *stats.Engine, tags ...stats.Tag) Metrics {
	return makeMetrics(eng, "read", "write", tags...)
}

func makeMetrics(eng *stats.Engine, reqOp string, resOp string, tags ...stats.Tag) Metrics {
	reqTags := make([]stats.Tag, 0, len(tags)+2)
	reqTags = append(reqTags, tags...)
	reqTags = append(reqTags, stats.Tag{"type", "request"}, stats.Tag{"operation", reqOp})

	resTags := make([]stats.Tag, 0, len(tags)+2)
	resTags = append(resTags, tags...)
	resTags = append(resTags, stats.Tag{"type", "response"}, stats.Tag{"operation", resOp})

	return Metrics{
		eng:     eng,
		reqTags: reqTags,
		resTags: resTags,
	}
}

func (m Metrics) ObserveRequest(req *http.Request) (body io.ReadCloser, tags []stats.Tag) {
	now := time.Now()

	tags = make([]stats.Tag, 0, len(m.reqTags)+10)
	tags = append(tags, m.reqTags...)
	tags = appendRequestTags(tags, req)

	rawTags := stats.MakeRawTags(tags)

	m.incrMessageCounter(tags, rawTags, now)
	m.observeHeaderSize(len(req.Header), tags, rawTags, now)
	m.observeHeaderLength(requestHeaderLength(req), tags, rawTags, now)

	body = newMessageBody(m, req.Body, tags, rawTags)
	return
}

func (m Metrics) ObserveResponse(res *http.Response) (body io.ReadCloser, tags []stats.Tag) {
	now := time.Now()

	tags = make([]stats.Tag, 0, len(m.resTags)+20)
	tags = append(tags, m.resTags...)
	tags = appendResponseTags(tags, res)

	if req := res.Request; req != nil {
		tags = appendRequestTags(tags, req)
	}

	rawTags := stats.MakeRawTags(tags)

	m.incrMessageCounter(tags, rawTags, now)
	m.observeHeaderSize(len(res.Header), tags, rawTags, now)
	m.observeHeaderLength(responseHeaderLength(res), tags, rawTags, now)

	body = newMessageBody(m, res.Body, tags, rawTags)
	return
}

func (m Metrics) incrErrorCounter(tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.errors.count"
	m.eng.Add(
		stats.CounterType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		1,
		now,
	)
}

func (m Metrics) incrMessageCounter(tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.message.count"
	m.eng.Add(
		stats.CounterType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		1,
		now,
	)
}

func (m Metrics) observeHeaderSize(size int, tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.message.header.sizes"
	m.eng.Observe(
		stats.HistogramType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		float64(size),
		now,
	)
}

func (m Metrics) observeHeaderLength(len int, tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.message.header.bytes"
	m.eng.Observe(
		stats.HistogramType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		float64(len),
		now,
	)
}

func (m Metrics) observeBodyLength(len int, tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.message.body.bytes"
	m.eng.Observe(
		stats.HistogramType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		float64(len),
		now,
	)
}

func (m Metrics) observeRTT(rtt time.Duration, tags []stats.Tag, rawTags stats.RawTags, now time.Time) {
	const name = "http.rtt.seconds"
	m.eng.Observe(
		stats.HistogramType,
		stats.RawMetricKey(name, rawTags),
		name,
		tags,
		rtt.Seconds(),
		now,
	)
}

func appendRequestTags(tags []stats.Tag, req *http.Request) []stats.Tag {
	ctype, charset := contentType(req.Header)
	return append(tags, // sorted
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
	return append(tags, // sorted
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

type messageBody struct {
	iostats.CountReader
	io.Closer
	metrics Metrics
	tags    []stats.Tag
	rawTags stats.RawTags
	once    sync.Once
}

func newMessageBody(metrics Metrics, body io.ReadCloser, tags []stats.Tag, rawTags stats.RawTags) *messageBody {
	return &messageBody{
		CountReader: iostats.CountReader{R: body},
		Closer:      body,
		metrics:     metrics,
		tags:        tags,
		rawTags:     rawTags,
	}
}

func (m *messageBody) Close() (err error) {
	err = m.Closer.Close()
	m.complete()
	return
}

func (m *messageBody) complete() {
	m.once.Do(func() { m.metrics.observeBodyLength(m.N, m.tags, m.rawTags, time.Now()) })
}

type nullBody struct{}

func (n *nullBody) Close() error { return nil }

func (n *nullBody) Read(b []byte) (int, error) { return 0, io.EOF }
