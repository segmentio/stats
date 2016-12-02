package httpstats

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/segmentio/stats/iostats"
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
		t.Run(fmt.Sprintf("%c", test.c), func(t *testing.T) {
			if is := isIDByte(test.c); is != test.is {
				t.Errorf("isIDByte(%c) != %t", test.c, test.is)
			}
		})
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
		t.Run(test.s, func(t *testing.T) {
			if is := isID(test.s); is != test.is {
				t.Errorf("isID(%s) != %t", test.s, test.is)
			}
		})
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
		t.Run(test.in, func(t *testing.T) {
			if out := sanitizeHttpPath(test.in); out != test.out {
				t.Errorf("sanitizeHttpPath(%s) => %s != %s", test.in, test.out, out)
			}
		})
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
		t.Run(strconv.Itoa(test.status), func(t *testing.T) {
			if s := responseStatusBucket(test.status); s != test.bucket {
				t.Errorf("bucket(%d) => %s != %s", test.status, test.bucket, s)
			}
		})
	}
}

func TestUrlLength(t *testing.T) {
	urllen := func(u *url.URL) int {
		return len(u.String())
	}

	newURL := func(s string) *url.URL {
		u, _ := url.Parse(s)
		return u
	}

	tests := []*url.URL{
		newURL("/"),
		newURL("[::1]:4242/hello/world"),
		newURL("http://localhost/?A=1&B=2"),
		newURL("http://luke:1234@localhost:4242/hello/world?A=1&B=2#space"),
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			n1 := urllen(test)
			n2 := urlLength(test)
			if n1 != n2 {
				t.Errorf("urlLength: %d != %d (%#v)", n1, n2, test)
			}
		})
	}
}

func TestHeaderLength(t *testing.T) {
	hdrlen := func(h1 http.Header) int {
		c := &iostats.CountWriter{W: ioutil.Discard}
		h2 := copyHeader(h1)
		h2.Write(c)
		return c.N + len("\r\n")
	}

	tests := []http.Header{
		http.Header{},
		http.Header{"Cookie": {}},
		http.Header{"Content-Type": {"application/json"}},
		http.Header{"Accept-Encoding": {"gzip", "deflate"}},
		http.Header{"Host": {"localhost"}, "Accept": {"text/html", "text/plan"}},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			n1 := hdrlen(test)
			n2 := headerLength(test)
			if n1 != n2 {
				t.Errorf("headerLength: %d != %d (%#v)", n1, n2, test)
			}
		})
	}
}

func TestRequestLength(t *testing.T) {
	reqlen := func(req *http.Request) int {
		c := &iostats.CountWriter{W: ioutil.Discard}
		r := &http.Request{
			Proto:            req.Proto,
			Method:           req.Method,
			URL:              req.URL,
			Host:             req.Host,
			ContentLength:    -1,
			TransferEncoding: req.TransferEncoding,
			Header:           copyHeader(req.Header),
			Body:             nopeReadCloser{},
		}
		if r.ContentLength >= 0 {
			r.Header.Set("Content-Length", strconv.FormatInt(r.ContentLength, 10))
		}
		r.Write(c)
		return c.N
	}

	tests := []*http.Request{
		&http.Request{
			Method:        "GET",
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			URL:           &url.URL{Path: "/"},
			Host:          "localhost",
			ContentLength: 11,
			Header:        http.Header{"Content-Type": {"text/plain"}},
			Body:          nopeReadCloser{},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			n1 := reqlen(test)
			n2 := requestHeaderLength(test)
			if n1 != n2 {
				t.Errorf("requestHeaderLength: %d != %d", n1, n2)
			}
		})
	}
}

func TestResponseLength(t *testing.T) {
	reslen := func(res *http.Response) int {
		c := &iostats.CountWriter{W: ioutil.Discard}
		r := &http.Response{
			StatusCode:       res.StatusCode,
			ProtoMajor:       res.ProtoMajor,
			ProtoMinor:       res.ProtoMinor,
			Request:          res.Request,
			TransferEncoding: res.TransferEncoding,
			Trailer:          res.Trailer,
			ContentLength:    -1,
			Header:           copyHeader(res.Header),
			Body:             nopeReadCloser{},
		}
		if r.ContentLength >= 0 {
			r.Header.Set("Content-Length", strconv.FormatInt(res.ContentLength, 10))
		}
		r.Write(c)
		return c.N
	}

	tests := []*http.Response{
		&http.Response{
			Proto:         "HTTP/1.1",
			StatusCode:    http.StatusOK,
			ProtoMajor:    1,
			ProtoMinor:    1,
			Request:       &http.Request{Method: "GET"},
			ContentLength: 11,
			Header:        http.Header{"Content-Type": {"text/plain"}},
			Body:          ioutil.NopCloser(strings.NewReader("Hello World!")),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			n1 := reslen(test)
			n2 := responseHeaderLength(test)
			if n1 != n2 {
				t.Errorf("responseHeaderLength: %d != %d", n1, n2)
			}
		})
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
		t.Run(test.host, func(t *testing.T) {
			if host, port := requestHost(test.req); host != test.host {
				t.Errorf("invalid request host: %#v != %#v", test.host, host)
			} else if port != test.port {
				t.Errorf("invalid request port: %#v != %#v", test.port, port)
			}
		})
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
		t.Run("", func(t *testing.T) {
			if te := transferEncoding(test.s); te != test.e {
				t.Errorf("invalid transfer encoding: %#v => %#v != %#v", test.s, test.e, te)
			}
		})
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
		t.Run(test.s, func(t *testing.T) {
			if ctype, charset := parseContentType(test.s); ctype != test.t {
				t.Errorf("invalid content type: %#v => %#v != %#v", test.s, test.t, ctype)
			} else if charset != test.c {
				t.Errorf("invalid charset: %#v => %#v != %v", test.s, test.c, charset)
			}
		})
	}
}
