package httpstats

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestCopyHeader(t *testing.T) {
	h1 := http.Header{"Content-Type": {"text/plain"}, "Content-Length": {"11"}}
	h2 := copyHeader(h1)

	if !reflect.DeepEqual(h1, h2) {
		t.Errorf("%v != %v", h1, h2)
	}
}

func TestResponseStatusBucket(t *testing.T) {
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
		if s := responseStatusBucket(test.status); s != test.bucket {
			t.Errorf("bucket(%d) => %s != %s", test.status, test.bucket, s)
		}
	}
}

func TestRequestLength(t *testing.T) {
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
		if n := requestHeaderLength(test.req); n != test.len {
			t.Errorf("requestHeaderLength #%d => %d != %d", i, test.len, n)
		}
	}
}

func TestResponseLength(t *testing.T) {
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
		if n := responseHeaderLength(test.res); n != test.len {
			t.Errorf("responseHeaderLength #%d => %d != %d", i, test.len, n)
		}
	}
}

func TestRequestHost(t *testing.T) {
	tests := []struct {
		req  *http.Request
		host string
	}{
		{
			req:  &http.Request{Host: "host"},
			host: "host",
		},
		{
			req:  &http.Request{URL: &url.URL{Host: "url"}},
			host: "url",
		},
		{
			req:  &http.Request{URL: &url.URL{}, Header: http.Header{"Host": {"header"}}},
			host: "header",
		},
	}

	for _, test := range tests {
		if host := requestHost(test.req); host != test.host {
			t.Errorf("invalid request host: %#v != %#v", test.host, host)
		}
	}
}
