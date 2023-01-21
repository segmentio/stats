package httpstats

import (
	"net/http"
	"time"

	"github.com/vertoforce/stats"
)

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

// RoundTrip implements http.RoundTripper
func (t *transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	start := time.Now()
	rtrip := t.transport
	eng := t.eng

	if rtrip == nil {
		rtrip = http.DefaultTransport
	}

	if tags := RequestTags(req); len(tags) > 0 {
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
	// safe guard, the transport should have done it already
	req.Body.Close() // nolint

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
