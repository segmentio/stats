package httpstats

import (
	"net/http"
	"time"

	"github.com/segmentio/stats"
)

const contextKeyReqTags = "segmentio_httpstats_req_tags"

// NewTransport wraps t to produce metrics on the default engine for every request
// sent and every response received.
func NewTransport(t http.RoundTripper) http.RoundTripper {
	return NewTransportWith(stats.DefaultEngine, t)
}

// NewTransportWith wraps t to produce metrics on eng for every request sent and
// every response received.
func NewTransportWith(eng *stats.Engine, t http.RoundTripper) http.RoundTripper {
	return &transport{
		transport: t,
		eng:       eng,
	}
}

type transport struct {
	transport http.RoundTripper
	eng       *stats.Engine
}

// RequestWithContext returns a shallow copy of req with its context changed with this provided tags
// so the they can be used later during the RoundTrip in the metrics recording.
// The provided ctx must be non-nil.
func RequestWithContext(req *http.Request, tags ...Tags) *http.Request {
	ctx := req.Context()
	ctx = ctx.WithValue(ctx, contextKeyReqTags, tags)
	return req.WithContext(ctx)
}

func (t *transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	start := time.Now()
	rtrip := t.transport
	eng := t.eng

	if rtrip == nil {
		rtrip = http.DefaultTransport
	}

	if tags, ok := req.Context().Value(contextKeyReqTags).([]stats.Tag); ok {
		eng = eng.WithTags(tags...)
	}

	if req.Body == nil {
		req.Body = &nullBody{}
	}

	m := &metrics{}

	req.Body = &requestBody{
		eng:     eng,
		req:     req,
		metrics: m,
		body:    req.Body,
		op:      "write",
	}

	res, err = rtrip.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		m.observeError(time.Now().Sub(start))
		eng.ReportAt(start, m)
	} else {
		res.Body = &responseBody{
			eng:     eng,
			res:     res,
			metrics: m,
			body:    res.Body,
			op:      "read",
			start:   start,
		}
	}

	return
}
