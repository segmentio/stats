package httpstats

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestIsIDByte(t *testing.T) {
	tests := []struct {
		c  byte
		is bool
	}{
		{'0', true},
		{'1', true},
		{'2', true},
		{'3', true},
		{'4', true},
		{'5', true},
		{'6', true},
		{'7', true},
		{'8', true},
		{'9', true},

		{'a', true},
		{'b', true},
		{'c', true},
		{'d', true},
		{'e', true},
		{'f', true},

		{'A', true},
		{'B', true},
		{'C', true},
		{'D', true},
		{'E', true},
		{'F', true},

		{'-', true},

		{'g', false},
		{'z', false},

		{'G', false},
		{'Z', false},

		{' ', false},
		{'!', false},
	}

	for _, test := range tests {
		if is := isIDByte(test.c); is != test.is {
			t.Errorf("isIDByte(%c) != %t", test.c, test.is)
		}
	}
}

func TestIsID(t *testing.T) {
	tests := []struct {
		s  string
		is bool
	}{
		{"0", true},
		{"1", true},
		{"1234567890", true},
		{"abcdef", true},
		{"ABCDEF", true},
		{"7CDACC74-F84B-4C2B-A4E0-7640A285F211", true},

		{"", false},
		{"Hello World!", false},
	}

	for _, test := range tests {
		if is := isID(test.s); is != test.is {
			t.Errorf("isID(%s) != %t", test.s, test.is)
		}
	}
}

func TestSanitizeHttpPath(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"/", "/"},
		{"/hello", "/hello"},
		{"/hello/world", "/hello/world"},
		{"/hello/1", "/hello/<id>"},
		{"/hello/7CDACC74-F84B-4C2B-A4E0-7640A285F211", "/hello/<id>"},
	}

	for _, test := range tests {
		if out := sanitizeHttpPath(test.in); out != test.out {
			t.Errorf("sanitizeHttpPath(%s) => %s != %s", test.in, test.out, out)
		}
	}
}

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
		port string
	}{
		{
			req:  &http.Request{Host: "host"},
			host: "host",
			port: "",
		},
		{
			req:  &http.Request{URL: &url.URL{Host: "url"}},
			host: "url",
			port: "",
		},
		{
			req:  &http.Request{URL: &url.URL{}, Header: http.Header{"Host": {"header:port"}}},
			host: "header",
			port: "port",
		},
	}

	for _, test := range tests {
		if host, port := requestHost(test.req); host != test.host {
			t.Errorf("invalid request host: %#v != %#v", test.host, host)
		} else if port != test.port {
			t.Errorf("invalid request port: %#v != %#v", test.port, port)
		}
	}
}

func TestTransferEncoding(t *testing.T) {
	tests := []struct {
		s []string
		e string
	}{
		{
			s: nil,
			e: "",
		},
		{
			s: []string{"chunked"},
			e: "chunked",
		},
		{
			s: []string{"chunked", "identity"},
			e: "chunked;identity",
		},
	}

	for _, test := range tests {
		if te := transferEncoding(test.s); te != test.e {
			t.Errorf("invalid transfer encoding: %#v => %#v != %#v", test.s, test.e, te)
		}
	}
}

func TestParseContentType(t *testing.T) {
	tests := []struct {
		s string
		t string
		c string
	}{
		{
			s: "",
			t: "",
			c: "",
		},
		{
			s: "text/html",
			t: "text/html",
			c: "",
		},
		{
			s: "text/html; charset=UTF-8",
			t: "text/html",
			c: "UTF-8",
		},
		{
			s: "text/html; charset=UTF-8;",
			t: "text/html",
			c: "UTF-8",
		},
		{
			s: "text/html;\ncharset=UTF-8\n",
			t: "text/html",
			c: "UTF-8",
		},
		{
			s: "charset=UTF-8",
			t: "",
			c: "UTF-8",
		},
	}

	for _, test := range tests {
		if ctype, charset := parseContentType(test.s); ctype != test.t {
			t.Errorf("invalid content type: %#v => %#v != %#v", test.s, test.t, ctype)
		} else if charset != test.c {
			t.Errorf("invalid charset: %#v => %#v != %v", test.s, test.c, charset)
		}
	}
}
