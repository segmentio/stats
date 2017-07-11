package redisstats

import (
	"time"

	"net"

	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
)

// NewHandler wraps h to produce metrics on the default engine for every request
// received and every response sent.
func NewHandler(h redis.Handler) redis.Handler {
	return NewHandlerWith(stats.DefaultEngine, h)
}

// NewHandlerWith wraps h to produce metrics on eng for every request received
// and every response sent.
func NewHandlerWith(engine *stats.Engine, h redis.Handler) redis.Handler {
	return &instrumentedRedisHandler{
		handler: h,
		engine:  engine,
	}
}

type instrumentedRedisHandler struct {
	handler redis.Handler
	engine  *stats.Engine
}

// ServeRedis implements the redis.Handler interface.
// It records a metric for each request.
func (h *instrumentedRedisHandler) ServeRedis(res redis.ResponseWriter, req *redis.Request) {
	h.engine.Incr("requests", stats.Tag{Name: "command", Value: req.Cmd})

	w := &errorWrappedResponseWriter{
		w: res,
	}

	startTime := time.Now()
	h.handler.ServeRedis(w, req)
	endTime := time.Now()

	switch e := w.err.(type) {
	case *net.OpError:
		h.engine.Incr("errors",
			stats.Tag{Name: "command", Value: req.Cmd},
			stats.Tag{Name: "type", Value: "network"},
			stats.Tag{Name: "operation", Value: e.Op})
	default:
		if e == nil {
			h.engine.ObserveDuration("request.rtt.seconds", endTime.Sub(startTime),
				stats.Tag{Name: "command", Value: req.Cmd})
		} else {
			h.engine.Incr("errors",
				stats.Tag{Name: "command", Value: req.Cmd})
		}
	}
}

type errorWrappedResponseWriter struct {
	w   redis.ResponseWriter
	err error
}

func (ew *errorWrappedResponseWriter) Write(v interface{}) error {
	err := ew.w.Write(v)
	ew.err = err
	return err
}

func (ew *errorWrappedResponseWriter) WriteStream(n int) error {
	err := ew.w.WriteStream(n)
	ew.err = err
	return err
}
