package httpstats

import (
	"io"
	"net/http"
	"sync"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/iostats"
)

func NewHandler(client stats.Client, handler http.Handler) http.Handler {
	return httpHandler{
		handler: handler,
		stats:   newHttpStats(client, "http_server"),
	}
}

type httpHandler struct {
	handler http.Handler
	stats   *httpStats
}

func (h httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	tags := httpRequestTags(req)
	clock := h.stats.timeReq.Start()
	clock.Stamp("read_header", tags...)

	c := io.Closer(req.Body)

	r := &iostats.CountReader{
		R: req.Body,
	}

	w := &httpResponseWriter{
		ResponseWriter: res,
		tags:           tags,
		clock:          clock,
	}

	once := &sync.Once{}
	body := func() { clock.Stamp("read_body", tags...) }

	req.Body = iostats.ReadCloser{
		Reader: iostats.ReaderFunc(func(b []byte) (n int, err error) {
			if n, err = r.Read(b); err == io.EOF {
				once.Do(body)
			}
			return
		}),
		Closer: iostats.CloserFunc(func() (err error) {
			err = c.Close()
			once.Do(body)
			return
		}),
	}

	res = w

	if _, ok := w.ResponseWriter.(http.Hijacker); ok {
		res = httpResponseHijacker{w}
	}

	h.stats.countReq.Add(1, tags...)
	h.handler.ServeHTTP(res, req)
	req.Body.Close()

	clock.Stamp("write_body", w.tags...)
	clock.Stop(w.tags...)

	h.stats.report(httpStatsReport{
		req: req,
		res: &http.Response{
			StatusCode:    w.status,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			ContentLength: -1,
			Request:       req,
			Header:        w.Header(),
			Body:          iostats.NopeReadCloser{},
		},
		reqBodyBytes: r.N,
		resBodyBytes: w.bytes,
		tags:         w.tags,
	})
}
