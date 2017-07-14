package redisstats

import (
	"bufio"
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
	return &handler{
		handler: h,
		engine:  engine,
	}
}

type handler struct {
	handler redis.Handler
	engine  *stats.Engine
}

// ServeRedis implements the redis.Handler interface.
// It records a metric for each request.
func (h *handler) ServeRedis(res redis.ResponseWriter, req *redis.Request) {
	h.engine.Incr("redis.request.count")
	h.engine.Observe("redis.request.size", float64(len(req.Cmds)))

	for _, cmd := range req.Cmds {
		tags := []stats.Tag{{"command", cmd.Cmd}}
		h.engine.Incr("redis.command.count", tags...)
		h.engine.Observe("redis.command_arguments.size", float64(argsLen(cmd.Args)), tags...)
	}

	w := &responseWriter{
		base: res,
	}

	startTime := time.Now()
	h.handler.ServeRedis(w, req)
	endTime := time.Now()
	rtt := endTime.Sub(startTime)

	for _, cmd := range req.Cmds {
		switch e := w.err.(type) {
		case *net.OpError:
			h.engine.Incr("redis.error.count",
				stats.Tag{"command", cmd.Cmd},
				stats.Tag{"operation", e.Op},
				stats.Tag{"type", "network"})

		default:
			h.engine.Incr("redis.error.count",
				stats.Tag{"command", cmd.Cmd})
		}

		tags := []stats.Tag{{"command", cmd.Cmd}}
		h.engine.ObserveDuration("redis.command.rtt.seconds", rtt, tags...)
	}
}

type responseWriter struct {
	base redis.ResponseWriter
	err  error
}

func (w *responseWriter) Write(v interface{}) error {
	err := w.base.Write(v)
	if err != nil && w.err == nil {
		w.err = err
	}
	return err
}

func (w *responseWriter) WriteStream(n int) error {
	err := w.base.WriteStream(n)
	if err != nil && w.err == nil {
		w.err = err
	}
	return err
}

func (w *responseWriter) Flush() (err error) {
	if f, ok := w.base.(redis.Flusher); ok {
		err = f.Flush()
	}
	if err != nil && w.err == nil {
		w.err = err
	}
	return nil
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.base.(redis.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, redis.ErrNotHijackable
}
