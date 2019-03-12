package httpstats

import (
	"net/http"
	"time"

	"github.com/segmentio/stats"
)

// Reporter is the interface for reporting stats at a given time
type Reporter interface {
	ReportAt(time time.Time, metrics interface{}, tags ...stats.Tag)
}

// NewTransport wraps t to produce metrics on the default engine for every request
// sent and every response received.
func NewTransport(t http.RoundTripper) http.RoundTripper {
	return NewTransportWith(stats.DefaultEngine, t)
}

// NewTransportWith wraps t to produce metrics on eng for every request sent and
// every response received.
func NewTransportWith(eng Reporter, t http.RoundTripper) http.RoundTripper {
	return &transport{
		transport: t,
		eng:       eng,
	}
}

type transport struct {
	transport http.RoundTripper
	eng       Reporter
}

func (t *transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	start := time.Now()
	rtrip := t.transport

	if rtrip == nil {
		rtrip = http.DefaultTransport
	}

	if req.Body == nil {
		req.Body = &nullBody{}
	}

	m := &metrics{}

	req.Body = &requestBody{
		eng:     t.eng,
		req:     req,
		metrics: m,
		body:    req.Body,
		op:      "write",
	}

	res, err = rtrip.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		m.observeError(time.Now().Sub(start))
		t.eng.ReportAt(start, m)
	} else {
		res.Body = &responseBody{
			eng:     t.eng,
			res:     res,
			metrics: m,
			body:    res.Body,
			op:      "read",
			start:   start,
		}
	}

	return
}
