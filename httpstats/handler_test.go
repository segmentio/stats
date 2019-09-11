package httpstats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestHandler(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

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

	measures := h.Measures()

	if len(measures) == 0 {
		t.Error("no measures reported by http handler")
	}

	for _, m := range measures {
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

	for _, m := range measures {
		t.Log(m)
	}
}

func TestDisableUserAgent(t *testing.T) {

	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	handler := NewHandlerWithConfig(&HandlerConfig{DisableUserAgent: true}, e, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ioutil.ReadAll(req.Body)
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("Hello World"))
	}))

	server := httptest.NewServer(handler)
	defer server.Close()

	userAgent := "Stats_UnitTests/1.0"
	client := &http.Client{}

	req, err := http.NewRequest("POST", server.URL, strings.NewReader("Hi"))
	if err != nil {
		t.Error(err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return
	}

	ioutil.ReadAll(res.Body)
	res.Body.Close()

	measures := h.Measures()

	if len(measures) == 0 {
		t.Error("no measures reported by http handler")
	}

	for _, m := range measures {
		for _, tag := range m.Tags {
			if tag.Name == "http_req_user_agent" {
				if tag.Value != "" {
					t.Errorf("expected user agent to be empty string. got:  %#v\n", tag.Value)
				}
			}
		}
	}

}

func TestHandlerHijack(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	server := httptest.NewServer(NewHandlerWith(e, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// make sure the response writer supports hijacking
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	})))
	defer server.Close()

	if _, err := http.Post(server.URL, "text/plain", strings.NewReader("Hi")); err == nil {
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
