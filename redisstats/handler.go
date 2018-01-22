package redisstats

import (
	"bufio"
	"reflect"
	"time"

	"net"

	"github.com/segmentio/objconv/resp"
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
	w := &responseWriter{base: res, count: 1, start: time.Now()}
	w.cmd = make([]commandMetrics, len(req.Cmds))
	w.req.request.count = 1
	w.req.request.size = len(req.Cmds)

	for i, cmd := range req.Cmds {
		c := &w.cmd[i]
		c.name = cmd.Cmd
		c.command.count = 1
		c.command.size = argsLen(cmd.Args)
	}

	h.handler.ServeRedis(w, req)
	rtt := time.Now().Sub(w.start)

	for i := range w.cmd {
		if c := &w.cmd[i]; c.command.rtt == 0 {
			c.command.rtt = rtt
		}
	}

	if errCount, errOp, errType := errorInfo(w.err); errCount != 0 {
		for i := range w.cmd {
			if c := &w.cmd[i]; c.error.count == 0 {
				c.error.count = errCount
				c.error.operation = errOp
				c.error.errtype = errType
			}
		}
	}

	h.engine.ReportAt(w.start, &w.req)
	h.engine.ReportAt(w.start, &w.cmd)
}

type responseWriter struct {
	base  redis.ResponseWriter
	start time.Time
	err   error
	index int
	count int
	metrics
}

func (w *responseWriter) Write(v interface{}) error {
	err := w.base.Write(v)

	if w.index < w.count {
		c := &w.cmd[w.index]

		if e, ok := err.(*resp.Error); ok {
			c.error.count = 1
			c.error.operation = "command"
			c.error.errtype = e.Type()
		}

		c.command.rtt = time.Now().Sub(w.start)
		c.command.rsize = valueLen(reflect.ValueOf(v))
		w.index++
	}

	return nil
}

func (w *responseWriter) WriteStream(n int) error {
	err := w.base.WriteStream(n)
	if err == nil {
		w.count = n
	} else if w.err == nil {
		w.err = err
	}
	return err
}

func (w *responseWriter) Flush() (err error) {
	if f, ok := w.base.(redis.Flusher); ok {
		if err = f.Flush(); err != nil && w.err == nil {
			w.err = err
		}
	}
	return nil
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.base.(redis.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, redis.ErrNotHijackable
}
