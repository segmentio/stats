package httpstats

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestTransport(t *testing.T) {
	type testRequest struct {
		req  *http.Request
		tags []stats.Tag
	}

	assertRequestTags := func(t *testing.T, actual []stats.Tag, expect []stats.Tag) {
		mapped := make(map[string]stats.Tag)
		for _, a := range actual {
			mapped[a.Name] = a
		}
		for _, e := range expect {
			if _, exists := mapped[e.Name]; !exists {
				t.Errorf("expected tag %s not found", e.Name)
			}
		}
	}

	newRequest := func(method string, path string, body io.Reader, tags ...stats.Tag) testRequest {
		req, _ := http.NewRequest(method, path, body)
		return testRequest{
			req:  req,
			tags: tags,
		}
	}

	for _, transport := range []http.RoundTripper{
		nil,
		&http.Transport{},
		http.DefaultTransport,
		http.DefaultClient.Transport,
	} {
		t.Run("", func(t *testing.T) {
			for _, reqCase := range []testRequest{
				newRequest("GET", "/", nil),
				newRequest("POST", "/", strings.NewReader("Hi")),
				newRequest("GET", "/", nil, stats.Tag{Name: "custom", Value: "perrequest"}),
			} {
				t.Run("", func(t *testing.T) {
					h := &statstest.Handler{}
					e := stats.NewEngine("", h)
					req := reqCase.req
					tags := reqCase.tags

					server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
						ioutil.ReadAll(req.Body)
						res.Write([]byte("Hello World!"))
					}))
					defer server.Close()

					httpc := &http.Client{
						Transport: NewTransportWith(e, transport),
					}

					req.URL.Scheme = "http"
					req.URL.Host = server.URL[7:]

					if len(tags) > 0 {
						req = RequestWithTags(req, tags...)
					}
					res, err := httpc.Do(req)
					if err != nil {
						t.Error(err)
						return
					}
					ioutil.ReadAll(res.Body)
					res.Body.Close()

					if len(h.Measures()) == 0 {
						t.Error("no measures reported by http handler")
					}

					for _, m := range h.Measures() {
						assertRequestTags(t, m.Tags, tags)
						for _, tag := range m.Tags {
							if tag.Name == "bucket" {
								switch tag.Value {
								case "2xx", "":
								default:
									t.Errorf("invalid bucket in measure event tags: %#v\n%#v", tag, m)
								}
							}
						}
					}
				})
			}
		})
	}
}

func TestTransportError(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	httpc := &http.Client{
		Transport: NewTransportWith(e, &http.Transport{}),
	}

	if _, err := httpc.Post(server.URL, "text/plain", strings.NewReader("Hi")); err == nil {
		t.Error("no error was reported by the http client")
	}

	measures := h.Measures()

	if len(measures) == 0 {
		t.Error("no measures reported by hijacked http handler")
	}

	for _, m := range measures {
		t.Log(m)
	}
}
