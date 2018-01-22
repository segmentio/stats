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
	newRequest := func(method string, path string, body io.Reader) *http.Request {
		req, _ := http.NewRequest(method, path, body)
		return req
	}

	for _, transport := range []http.RoundTripper{
		nil,
		&http.Transport{},
		http.DefaultTransport,
		http.DefaultClient.Transport,
	} {
		t.Run("", func(t *testing.T) {
			for _, req := range []*http.Request{
				newRequest("GET", "/", nil),
				newRequest("POST", "/", strings.NewReader("Hi")),
			} {
				t.Run("", func(t *testing.T) {
					h := &statstest.Handler{}
					e := stats.NewEngine("", h)

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
