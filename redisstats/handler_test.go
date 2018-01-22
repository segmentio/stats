package redisstats

import (
	"errors"
	"testing"

	"net"

	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
)

func TestHandler(t *testing.T) {
	statsHandler := &statstest.Handler{}
	e := stats.NewEngine("", statsHandler)

	h := NewHandlerWith(e, &testRedisHandler{})

	h.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))

	h.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo", "bar")))

	h.ServeRedis(&testResponseWriter{err: &net.OpError{Op: "read"}},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo")))

	h.ServeRedis(&testResponseWriter{err: errors.New("fail")},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))

	measures := statsHandler.Measures()
	if len(measures) == 0 {
		t.Error("no measures were produced")
	}

	for _, m := range measures {
		t.Log(m)
	}
}

type testRedisHandler struct{}

func (*testRedisHandler) ServeRedis(res redis.ResponseWriter, req *redis.Request) {
	res.Write("OK")
}

type testResponseWriter struct {
	err error
}

func (w *testResponseWriter) WriteStream(n int) error {
	return w.err
}

func (w *testResponseWriter) Write(v interface{}) error {
	// Discard the data; we don't care
	return w.err
}
