package httpstats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/segmentio/stats"
)

func TestTransport(t *testing.T) {
	for _, transport := range []http.RoundTripper{
		&http.Transport{},
		http.DefaultTransport,
		http.DefaultClient.Transport,
	} {
		backend := &stats.EventBackend{}
		client := stats.NewClient(backend)
		defer client.Close()

		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			ioutil.ReadAll(req.Body)
			res.Write([]byte("Hello World!"))
		}))
		defer server.Close()

		httpc := &http.Client{
			Transport: NewTransport(client, transport),
		}

		res, err := httpc.Post(server.URL, "text/plain", strings.NewReader("Hi"))
		if err != nil {
			t.Error(err)
			return
		}
		ioutil.ReadAll(res.Body)
		res.Body.Close()

		backend.RLock()
		defer backend.RUnlock()

		if len(backend.Events) == 0 {
			t.Error("no metric events were produced by the http transport")
		}

		for _, e := range backend.Events {
			switch s := e.Tags.Get("bucket"); s {
			case "2xx", "":
			default:
				t.Errorf("invalid bucket in metric event tags: %s\n%v", s, e)
			}
		}
	}
}

func TestTransportError(t *testing.T) {
	backend := &stats.EventBackend{}
	client := stats.NewClient(backend)
	defer client.Close()

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	httpc := &http.Client{
		Transport: NewTransport(client, &http.Transport{}),
	}

	if _, err := httpc.Post(server.URL, "text/plain", strings.NewReader("Hi")); err == nil {
		t.Error("no error was reported by the http client")
	}

	backend.RLock()
	defer backend.RUnlock()

	if len(backend.Events) == 0 {
		t.Error("no metric events were produced by the http transport")
	}
}
