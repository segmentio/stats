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
	backend := &stats.EventBackend{}
	client := stats.NewClient("", backend)
	defer client.Close()

	server := httptest.NewServer(NewHandler(client, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ioutil.ReadAll(req.Body)
		res.Write([]byte("Hello World"))

		// make sure the response writer supports hijacking
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	})))
	defer server.Close()

	res, err := http.Post(server.URL, "text/plain", strings.NewReader("Hi"))
	if err != nil {
		t.Error(err)
		return
	}
	ioutil.ReadAll(res.Body)
	res.Body.Close()

	backend.RLock()
	defer backend.RUnlock()

	if len(backend.Events) == 0 {
		t.Error("no metric events were produced by the http handler")
	}

	for _, e := range backend.Events {
		switch s := e.Tags.Get("bucket"); s {
		case "2xx", "":
		default:
			t.Errorf("invalid bucket in metric event tags: %s\n%v", s, e)
		}
	}
}
