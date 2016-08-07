package stats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestCopyHttpHeader(t *testing.T) {
	h1 := http.Header{"Content-Type": {"text/plain"}, "Content-Length": {"11"}}
	h2 := copyHttpHeader(h1)

	if !reflect.DeepEqual(h1, h2) {
		t.Errorf("%v != %v", h1, h2)
	}
}

func TestHttpResponseStatusBucket(t *testing.T) {
	tests := []struct {
		status int
		bucket string
	}{
		{
			status: 0,
			bucket: "???",
		},
		{
			status: 100,
			bucket: "1xx",
		},
		{
			status: 200,
			bucket: "2xx",
		},
		{
			status: 300,
			bucket: "3xx",
		},
		{
			status: 400,
			bucket: "4xx",
		},
		{
			status: 500,
			bucket: "5xx",
		},
	}

	for _, test := range tests {
		if s := httpResponseStatusBucket(test.status); s != test.bucket {
			t.Errorf("bucket(%d) => %s != %s", test.status, test.bucket, s)
		}
	}
}

func TestHttpRequestLength(t *testing.T) {
	tests := []struct {
		req *http.Request
		len int
	}{
		{
			req: &http.Request{
				Method:        "GET",
				URL:           &url.URL{Path: "/"},
				Host:          "localhost",
				ContentLength: 11,
				Header:        http.Header{"Content-Type": {"text/plain"}},
				Body:          ioutil.NopCloser(strings.NewReader("Hello World!")),
			},
			len: 93,
		},
	}

	for i, test := range tests {
		if n := httpRequestHeaderLength(test.req); n != test.len {
			t.Errorf("httpRequestHeaderLength #%d => %d != %d", i, test.len, n)
		}
	}
}

func TestHttpResponseLength(t *testing.T) {
	tests := []struct {
		res *http.Response
		len int
	}{
		{
			res: &http.Response{
				StatusCode:    http.StatusOK,
				ProtoMajor:    1,
				ProtoMinor:    1,
				Request:       &http.Request{Method: "GET"},
				ContentLength: 11,
				Header:        http.Header{"Content-Type": {"text/plain"}},
				Body:          ioutil.NopCloser(strings.NewReader("Hello World!")),
			},
			len: 64,
		},
	}

	for i, test := range tests {
		if n := httpResponseHeaderLength(test.res); n != test.len {
			t.Errorf("httpResponseHeaderLength #%d => %d != %d", i, test.len, n)
		}
	}
}

func TestHttpHandler(t *testing.T) {
	backend := &EventBackend{}
	client := NewClient("", backend)
	defer client.Close()

	server := httptest.NewServer(NewHttpHandler(client, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
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

func TestHttpRoundTripper(t *testing.T) {
	backend := &EventBackend{}
	client := NewClient("", backend)
	defer client.Close()

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ioutil.ReadAll(req.Body)
		res.Write([]byte("Hello World!"))
	}))
	defer server.Close()

	httpc := &http.Client{
		Transport: NewHttpTransport(client, &http.Transport{}),
	}

	res, err := httpc.Post(server.URL, "text/plain", strings.NewReader("Hi"))
	if err != nil {
		t.Error(err)
		return
	}
	ioutil.ReadAll(res.Body)
	res.Body.Close()

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

func TestHttpRoundTripperError(t *testing.T) {
	backend := &EventBackend{}
	client := NewClient("", backend)
	defer client.Close()

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		conn, _, _ := res.(http.Hijacker).Hijack()
		conn.Close()
	}))
	defer server.Close()

	httpc := &http.Client{
		Transport: NewHttpTransport(client, &http.Transport{}),
	}

	_, err := httpc.Post(server.URL, "text/plain", strings.NewReader("Hi"))
	if err == nil {
		t.Error("no error was reported by the http client")
	}

	if len(backend.Events) == 0 {
		t.Error("no metric events were produced by the http transport")
	}
}
