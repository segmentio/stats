package redisstats

import (
	"errors"
	"testing"

	"net"

	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
)

func TestHandler(t *testing.T) {
	metricHandler := &testMetricHandler{}
	e := stats.NewEngine("")
	e.Register(metricHandler)

	wrappedRedisHandler := NewHandlerWith(e, &testRedisHandler{})

	wrappedRedisHandler.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))
	wrappedRedisHandler.ServeRedis(&testResponseWriter{},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo", "bar")))

	wrappedRedisHandler.ServeRedis(&testResponseWriter{err: &net.OpError{Op: "read"}},
		redis.NewRequest("127.0.0.1:6379", "GET", redis.List("foo")))

	wrappedRedisHandler.ServeRedis(&testResponseWriter{err: errors.New("fail")},
		redis.NewRequest("127.0.0.1:6379", "SET", redis.List("foo", "bar")))

	if len(metricHandler.metrics) == 0 {
		t.Error("no metrics reported by http handler")
	}

	validateMetric(t, metricHandler.metrics[0], "commands",
		[]stats.Tag{{Name: "name", Value: "SET"}}, stats.CounterType)

	validateMetric(t, metricHandler.metrics[1], "roundTripTime",
		[]stats.Tag{{Name: "command", Value: "SET"}}, stats.HistogramType)

	validateMetric(t, metricHandler.metrics[2], "commands",
		[]stats.Tag{{Name: "name", Value: "GET"}}, stats.CounterType)

	validateMetric(t, metricHandler.metrics[3], "roundTripTime",
		[]stats.Tag{{Name: "command", Value: "GET"}}, stats.HistogramType)

	validateMetric(t, metricHandler.metrics[4], "commands",
		[]stats.Tag{{Name: "name", Value: "GET"}}, stats.CounterType)

	validateMetric(t, metricHandler.metrics[5], "errors",
		[]stats.Tag{
			{Name: "command", Value: "GET"},
			{Name: "type", Value: "network"},
			{Name: "operation", Value: "read"},
		}, stats.CounterType)

	validateMetric(t, metricHandler.metrics[6], "commands",
		[]stats.Tag{{Name: "name", Value: "SET"}}, stats.CounterType)

	validateMetric(t, metricHandler.metrics[7], "errors",
		[]stats.Tag{
			{Name: "command", Value: "SET"},
		}, stats.CounterType)
}

type testRedisHandler struct{}

func (*testRedisHandler) ServeRedis(res redis.ResponseWriter, req *redis.Request) {
	res.Write("+OK\r\n")
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
