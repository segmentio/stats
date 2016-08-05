package prometheus

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHttpHandler(t *testing.T) {
	server := httptest.NewServer(NewHttpHandler(HandlerFunc(func(res ResponseWriter, req *http.Request) {
		res.Header().Set("Server", "test")

		res.WriteMetric(Metric{
			Type:   "gauge",
			Name:   "hello_world",
			Help:   "this is a gauge",
			Value:  1,
			Labels: Labels{{"hello", "world"}},
		})

		res.WriteMetric(Metric{
			Name:  "hello_world",
			Value: 10,
		})

		res.WriteMetric(Metric{
			Type:  "counter",
			Name:  "hello_you",
			Help:  "this is a counter",
			Value: 42,
			Time:  time.Unix(1, 0),
		})
	})))
	defer server.Close()

	res, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}

	if s := string(content); s != `# HELP hello_world this is a gauge
# TYPE hello_world gauge
hello_world{hello="world"} 1
hello_world 10

# HELP hello_you this is a counter
# TYPE hello_you counter
hello_you 42 1000
` {
		t.Error(s)
	}
}

func TestHttpHandlerRedirect(t *testing.T) {
	server := httptest.NewServer(NewHttpHandler(HandlerFunc(func(res ResponseWriter, req *http.Request) {
		res.WriteMetric(Metric{
			Type:   "gauge",
			Name:   "hello_world",
			Help:   "this is a gauge",
			Value:  1,
			Labels: Labels{{"hello", "world"}},
		})
	})))
	defer server.Close()

	res, err := http.Get(server.URL + "/")
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
		return
	}

	if s := string(content); s != `# HELP hello_world this is a gauge
# TYPE hello_world gauge
hello_world{hello="world"} 1
` {
		t.Error(s)
	}
}

func TestHttpHandlerMethodNotAllowed(t *testing.T) {
	server := httptest.NewServer(NewHttpHandler(HandlerFunc(func(res ResponseWriter, req *http.Request) {
		t.Error("the handler should never be called")
	})))
	defer server.Close()

	res, err := http.Post(server.URL+"/metrics", "text/plain", strings.NewReader(""))
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("only GET and HED should be allowed")
	}
}
