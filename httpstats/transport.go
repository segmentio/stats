package httpstats

import (
	"net/http"

	"github.com/segmentio/stats"
)

func NewTransport(eng *stats.Engine, transport http.RoundTripper, tags ...stats.Tag) http.RoundTripper {
	return &httpTransport{
		transport: transport,
		metrics:   MakeClientMetrics(eng, tags...),
	}
}

type httpTransport struct {
	transport http.RoundTripper
	metrics   Metrics
}

func (t *httpTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var clock = t.metrics.RTT.Start()
	var tags []stats.Tag

	if t.transport == nil {
		t.transport = http.DefaultTransport
	}

	req.Body, tags = t.metrics.ObserveRequest(req)
	res, err = t.transport.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		t.metrics.Errors.Clone(tags...).Incr()
		clock.Clone(tags...).Stop()
	} else {
		res.Body, _ = t.metrics.ObserveResponse(res, clock)
	}

	return
}
