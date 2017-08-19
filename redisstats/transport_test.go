package redisstats

import (
	"net"
	"testing"

	"github.com/segmentio/objconv/resp"
	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestTransport(t *testing.T) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	client := redis.Client{
		Addr:      "127.0.0.1",
		Transport: NewTransportWith(e, &testTransport{}),
	}

	respErrorClient := redis.Client{
		Addr:      "127.0.0.1",
		Transport: NewTransportWith(e, &testTransport{err: resp.NewError("request failed")}),
	}

	readErrorClient := redis.Client{
		Addr:      "127.0.0.1",
		Transport: NewTransportWith(e, &testTransport{err: &net.OpError{Op: "read"}}),
	}

	writeErrorClient := redis.Client{
		Addr:      "127.0.0.1",
		Transport: NewTransportWith(e, &testTransport{err: &net.OpError{Op: "write"}}),
	}

	var res *redis.Response

	res, _ = client.Do(redis.NewRequest("1.2.3.4:6379", "SET", redis.List("foo", "bar")))
	res.Close()

	res, _ = client.Do(redis.NewRequest("5.6.7.8:6379", "GET", redis.List("foo")))
	res.Close()

	respErrorClient.Do(redis.NewRequest("9.9.9.9:6379", "GET", redis.List("foo")))
	readErrorClient.Do(redis.NewRequest("9.9.9.9:6379", "GET", redis.List("foo")))
	writeErrorClient.Do(redis.NewRequest("9.9.9.9:6379", "SET", redis.List("foo", "bar")))

	measures := h.Measures()
	if len(measures) == 0 {
		t.Error("no measures were produced")
	}

	for _, m := range measures {
		t.Log(m)
	}
}

type testTransport struct {
	err error
}

func (tr *testTransport) RoundTrip(req *redis.Request) (*redis.Response, error) {
	defer func() {
		for _, cmd := range req.Cmds {
			if cmd.Args != nil {
				cmd.Args.Close()
			}
		}
	}()
	if tr.err != nil {
		return nil, tr.err
	}
	return &redis.Response{Args: redis.List("OK"), Request: req}, nil
}
