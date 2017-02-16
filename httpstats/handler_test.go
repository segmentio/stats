package httpstats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/segmentio/stats"
)

func TestHandler(t *testing.T) {
	h := &metricHandler{}
	e := stats.NewEngine()
	e.Register(h)

	server := httptest.NewServer(NewHandlerWith(e, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ioutil.ReadAll(req.Body)
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("Hello World"))
	})))
	defer server.Close()

	res, err := http.Post(server.URL, "text/plain", strings.NewReader("Hi"))
	if err != nil {
		t.Error(err)
		return
	}
	ioutil.ReadAll(res.Body)
	res.Body.Close()

	if len(h.metrics) == 0 {
		t.Error("no metrics reported by http handler")
	}

	for _, m := range h.metrics {
		for _, tag := range m.Tags {
			if tag.Name == "bucket" {
				switch tag.Value {
				case "2xx", "":
				default:
					t.Errorf("invalid bucket in metric event tags: %#v\n%#v", tag, m)
				}
			}
		}
	}
}

func TestHandlerHijack(t *testing.T) {
	h := &metricHandler{}
	e := stats.NewEngine()
	e.Register(h)

	server := httptest.NewServer(NewHandlerWith(e, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// make sure the response writer supports hijacking
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	})))
	defer server.Close()

	if _, err := http.Post(server.URL, "text/plain", strings.NewReader("Hi")); err == nil {
		t.Error("no error was reported by the http client")
	}

	if len(h.metrics) == 0 {
		t.Error("no metrics reported by hijacked http handler")
	}
}
