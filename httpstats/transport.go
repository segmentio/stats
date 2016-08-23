package httpstats

import (
	"net/http"

	"github.com/segmentio/stats"
)

func NewTransport(client stats.Client, transport http.RoundTripper) http.RoundTripper {
	return httpTransport{
		transport: transport,
		metrics:   NewClientMetrics(client),
	}
}

type httpTransport struct {
	transport http.RoundTripper
	metrics   *Metrics
}

func (t httpTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	var clock stats.Clock

	clock, req.Body = t.metrics.ObserveRequest(req)
	res, err = t.transport.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already
	clock.Stop()

	if err != nil {
		t.metrics.Errors.Add(1, makeRequestTags(req)...)
		clock.Stop()
	} else {
		res.Body = t.metrics.ObserveResponse(res, clock)
	}

	return
}
