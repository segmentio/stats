package httpstats

import (
	"io"
	"net/http"
	"sync"
	"time"

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
	var tags []stats.Tag
	var start = time.Now()

	if t.transport == nil {
		t.transport = http.DefaultTransport
	}

	if req.Body == nil {
		req.Body = nullBody{}
	}

	req.Body, tags = t.metrics.ObserveRequest(req)
	res, err = t.transport.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		t.metrics.incrErrorCounter(tags, stats.MakeRawTags(tags), start)
	}

	if res != nil {
		body, tags := t.metrics.ObserveResponse(res)
		res.Body = &httpTransportResponseBody{
			ReadCloser: body,
			metrics:    t.metrics,
			tags:       tags,
			start:      start,
		}
	}

	return
}

type httpTransportResponseBody struct {
	io.ReadCloser
	metrics Metrics
	tags    []stats.Tag
	start   time.Time
	once    sync.Once
}

func (t *httpTransportResponseBody) Close() (err error) {
	err = t.ReadCloser.Close()
	t.once.Do(t.complete)
	return
}

func (t *httpTransportResponseBody) complete() {
	now := time.Now()
	t.metrics.observeRTT(now.Sub(t.start), t.tags, stats.MakeRawTags(t.tags), now)
}
