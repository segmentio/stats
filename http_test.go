package stats

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHttpHandler(t *testing.T) {
	tests := []struct {
		status int
		events int
		bucket string
		body   string
	}{
		{
			status: 101,
			events: 11,
			bucket: "1xx",
			body:   "Hello World!",
		},
		{
			status: 200,
			events: 11,
			bucket: "2xx",
			body:   "Hello World!",
		},
		{
			status: 300,
			events: 11,
			bucket: "3xx",
			body:   "Hello World!",
		},
		{
			status: 400,
			events: 11,
			bucket: "4xx",
			body:   "Hello World!",
		},
		{
			status: 500,
			events: 11,
			bucket: "5xx",
			body:   "Hello World!",
		},
		{
			status: 600,
			events: 11,
			bucket: "???",
			body:   "Hello World!",
		},
	}

	for _, test := range tests {
		events, body, err := runTestHttpHandler(test.status, test.body)

		if err != nil {
			t.Error(err)
			continue
		}

		if test.status >= 200 && body != test.body {
			t.Errorf("test-status-%d: invalid response body: %#v != %#v", test.status, test.body, body)
			continue
		}

		if len(events) != test.events {
			t.Errorf("test-status-%d: invalid metric events, expected %d but found %d:", test.status, test.events, len(events))
		}

		for i, e := range events {
			if i != 0 {
				if bucket := e.Tags.Get("bucket"); bucket != test.bucket {
					t.Errorf("tests-status-%d: event %# has an invalid bucket, %#v != %#v\n- %v", test.status, i, test.bucket, bucket, e)
				}
			}
		}
	}
}

func runTestHttpHandler(status int, body string) (events []Event, response string, err error) {
	backend := &EventBackend{}
	client := NewClient("", backend)
	defer client.Close()

	server := httptest.NewServer(NewHttpHandler(client, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(status)
		io.Copy(res, req.Body)
	})))
	defer server.Close()
	var res *http.Response

	if res, err = http.Post(server.URL, "text/plain", strings.NewReader(body)); err != nil {
		return
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	events, response = backend.Events, string(b)
	return
}
