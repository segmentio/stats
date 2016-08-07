package httpstats

import (
	"io"
	"net/http"
	"sync"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/iostats"
)

func NewTransport(client stats.Client, transport http.RoundTripper) http.RoundTripper {
	return httpTransport{
		transport: transport,
		stats:     newHttpStats(client, "http_client"),
	}
}

type httpTransport struct {
	transport http.RoundTripper
	stats     *httpStats
}

func (t httpTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	tags := httpRequestTags(req)
	clock := t.stats.timeReq.Start()

	r := &httpStatsReport{
		req:  req,
		tags: tags,
	}

	t.setup(r, req, clock)

	res, err = t.transport.RoundTrip(req)
	req.Body.Close() // safe guard, the transport should have done it already

	if err != nil {
		t.teardownWithError(r, err, clock)
	} else {
		t.teardownWithResponse(r, res, clock)
	}

	return
}

func (t httpTransport) setup(report *httpStatsReport, req *http.Request, clock stats.Clock) {
	r := &iostats.CountReader{R: req.Body}
	c := io.Closer(req.Body)

	once1 := &sync.Once{}
	once2 := &sync.Once{}

	do1 := func() { clock.Stamp("write_header", report.tags...) }
	do2 := func() { clock.Stamp("write_body", report.tags...) }

	req.Body = iostats.ReadCloser{
		Reader: iostats.ReaderFunc(func(b []byte) (int, error) {
			once1.Do(do1)
			return r.Read(b)
		}),
		Closer: iostats.CloserFunc(func() (err error) {
			err = c.Close()
			once1.Do(do1)
			once2.Do(do2)
			report.reqBodyBytes = r.N
			return
		}),
	}

	return
}

func (t httpTransport) teardownWithError(report *httpStatsReport, err error, clock stats.Clock) {
	report.err = err
	clock.Stop(report.tags...)
	t.stats.report(*report)
}

func (t httpTransport) teardownWithResponse(report *httpStatsReport, res *http.Response, clock stats.Clock) {
	report.res = res
	report.tags = append(report.tags, httpResponseTags(res.StatusCode, res.Header)...)
	clock.Stamp("read_header", report.tags...)

	r := &iostats.CountReader{R: res.Body}
	c := io.Closer(res.Body)

	once1 := &sync.Once{}
	once2 := &sync.Once{}

	do := func() { clock.Stamp("read_body", report.tags...) }

	res.Body = iostats.ReadCloser{
		Reader: iostats.ReaderFunc(func(b []byte) (n int, err error) {
			if n, err = r.Read(b); err == io.EOF {
				once1.Do(do)
			}
			return
		}),
		Closer: iostats.CloserFunc(func() (err error) {
			err = c.Close()
			once1.Do(do)
			once2.Do(func() {
				clock.Stop(report.tags...)
				report.err = err
				report.resBodyBytes = r.N
				t.stats.report(*report)
			})
			return
		}),
	}
}
