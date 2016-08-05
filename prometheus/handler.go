package prometheus

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type ResponseWriter interface {
	http.ResponseWriter

	MetricWriter
}

type Handler interface {
	HandleMetrics(res ResponseWriter, req *http.Request)
}

type HandlerFunc func(ResponseWriter, *http.Request)

func (f HandlerFunc) HandleMetrics(res ResponseWriter, req *http.Request) { f(res, req) }

func NewHttpHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) { serveHTTP(h, res, req) })
}

func serveHTTP(h Handler, res http.ResponseWriter, req *http.Request) {
	hdr := res.Header()
	hdr.Set("Server", "github.com/segmentio/stats/prometheus")
	hdr.Set("Content-Type", "text/plain; version=0.0.4")

	switch req.Method {
	case "GET", "HEAD":
	default:
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

acceptEncoding:
	for _, encoding := range split(req.Header.Get("Accept-Encoding")) {
		switch encoding {
		case "gzip":
			res = newGzipResponseWriter(res)
			hdr.Set("Content-Encoding", "gzip")
			break acceptEncoding
		}
	}

	req.Header.Del("Accept")
	req.Header.Del("Accept-Encoding")

	switch req.URL.Path {
	case "/metrics":
		h.HandleMetrics(struct {
			http.ResponseWriter
			MetricWriter
		}{res, NewTextWriter(res)}, req)

	default:
		hdr.Set("Location", "/metrics")
		res.WriteHeader(http.StatusMovedPermanently)
	}

	if c, ok := res.(io.Closer); ok {
		c.Close()
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	*gzip.Writer
}

func (w gzipResponseWriter) Header() http.Header         { return w.ResponseWriter.Header() }
func (w gzipResponseWriter) Write(b []byte) (int, error) { return w.Writer.Write(b) }

func newGzipResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	z, _ := gzip.NewWriterLevel(w, gzip.BestCompression)
	return gzipResponseWriter{w, z}
}

func split(s string) []string {
	parts := strings.Split(s, ",")

	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}

	return parts
}
