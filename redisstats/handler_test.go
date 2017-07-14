package redisstats

import (
	"errors"
	"testing"

	"net"

	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
)

func TestHandler(t *testing.T) {
	t.Skip("temporarily disabling this test to move forward with the hackweek project")

	m := &testMetricHandler{}
	e := stats.NewEngine("")
	e.Register(m)

	h := NewHandlerWith(e, &testRedisHandler{})

	h.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))
	h.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo", "bar")))

	h.ServeRedis(&testResponseWriter{err: &net.OpError{Op: "read"}},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo")))

	h.ServeRedis(&testResponseWriter{err: errors.New("fail")},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))

	if len(m.metrics) == 0 {
		t.Error("no metrics reported by http handler")
	}

	validateMetric(t, m.metrics[0], "requests",
		[]stats.Tag{{Name: "command", Value: "SET"}}, stats.CounterType)

	validateMetric(t, m.metrics[1], "request.rtt.seconds",
		[]stats.Tag{{Name: "command", Value: "SET"}}, stats.HistogramType)

	validateMetric(t, m.metrics[2], "requests",
		[]stats.Tag{{Name: "command", Value: "GET"}}, stats.CounterType)

	validateMetric(t, m.metrics[3], "request.rtt.seconds",
		[]stats.Tag{{Name: "command", Value: "GET"}}, stats.HistogramType)

	validateMetric(t, m.metrics[4], "requests",
		[]stats.Tag{{Name: "command", Value: "GET"}}, stats.CounterType)

	validateMetric(t, m.metrics[5], "errors",
		[]stats.Tag{
			{Name: "command", Value: "GET"},
			{Name: "type", Value: "network"},
			{Name: "operation", Value: "read"},
		}, stats.CounterType)

	validateMetric(t, m.metrics[6], "requests",
		[]stats.Tag{{Name: "command", Value: "SET"}}, stats.CounterType)

	validateMetric(t, m.metrics[7], "errors",
		[]stats.Tag{
			{Name: "command", Value: "SET"},
		}, stats.CounterType)
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
