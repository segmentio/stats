package redisstats

import (
	"sync"
	"time"

	"net"

	"github.com/segmentio/objconv/resp"
	"github.com/segmentio/redis-go"
	"github.com/segmentio/stats"
)

// NewTransport wraps rt to produce metrics on the default engine for every request
// received and every response sent.
func NewTransport(rt redis.RoundTripper) redis.RoundTripper {
	return NewTransportWith(stats.DefaultEngine, rt)
}

// NewTransportWith wraps t to produce metrics on eng for every request received
// and every response sent.
func NewTransportWith(engine *stats.Engine, rt redis.RoundTripper) redis.RoundTripper {
	return &transport{
		rt:     rt,
		engine: engine,
	}
}

type transport struct {
	rt     redis.RoundTripper
	engine *stats.Engine
}

// RoundTrip performs the request req, and returns the associated response.
func (t *transport) RoundTrip(req *redis.Request) (*redis.Response, error) {
	t.engine.Incr("redis.request.count")
	t.engine.Observe("redis.request.size", float64(len(req.Cmds)))

	for _, cmd := range req.Cmds {
		tags := []stats.Tag{{"command", cmd.Cmd}}
		t.engine.Incr("redis.command.count", tags...)
		t.engine.Observe("redis.command_arguments.size", float64(argsLen(cmd.Args)), tags...)
	}

	startTime := time.Now()
	res, err := t.rt.RoundTrip(req)

	if res != nil && res.Args != nil {
		res.Args = &args{
			Args:   res.Args,
			start:  startTime,
			req:    req,
			engine: t.engine,
		}
	}

	if res != nil && res.TxArgs != nil {
		res.TxArgs = &txArgs{
			TxArgs: res.TxArgs,
			start:  startTime,
			req:    req,
			engine: t.engine,
		}
	}

	return res, err
}

type args struct {
	redis.Args
	start  time.Time
	once   sync.Once
	req    *redis.Request
	engine *stats.Engine
}

func (a *args) Close() error {
	err := a.Args.Close()
	a.once.Do(func() { observeResponse(a.engine, a.req, err, time.Now().Sub(a.start)) })
	return err
}

type txArgs struct {
	redis.TxArgs
	start  time.Time
	once   sync.Once
	req    *redis.Request
	engine *stats.Engine
}

func (a *txArgs) Close() error {
	err := a.TxArgs.Close()
	a.once.Do(func() { observeResponse(a.engine, a.req, err, time.Now().Sub(a.start)) })
	return err
}

func observeResponse(eng *stats.Engine, req *redis.Request, err error, rtt time.Duration) {
	for _, cmd := range req.Cmds {
		switch e := err.(type) {
		case *net.OpError:
			eng.Incr("redis.error.count",
				stats.Tag{"command", cmd.Cmd},
				stats.Tag{"operation", e.Op},
				stats.Tag{"type", "network"})

		case *resp.Error:
			eng.Incr("redis.error.count",
				stats.Tag{"command", cmd.Cmd},
				stats.Tag{"type", e.Type()})

		default:
			eng.Incr("redis.error.count",
				stats.Tag{"command", cmd.Cmd})
		}

		tags := []stats.Tag{{"command", cmd.Cmd}}
		eng.ObserveDuration("redis.command.rtt.seconds", rtt, tags...)
	}
}

func argsLen(args redis.Args) int {
	if args == nil {
		return 0
	}
	return args.Len()
}
