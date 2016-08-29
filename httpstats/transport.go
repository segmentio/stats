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
	var clock = t.metrics.RTT.Start()
	var tags stats.Tags

	if t.transport == nil {
		t.transport = http.DefaultTransport
	}

	req.Body, tags = t.metrics.ObserveRequest(req)
	res, err = t.transport.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		t.metrics.Errors.Add(1, makeRequestTags(req)...)
		clock.Stop(tags...)
	} else {
		res.Body, _ = t.metrics.ObserveResponse(res, clock)
	}

	return
}
