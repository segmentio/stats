package redisstats

import (
	"reflect"
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
	start := time.Now()

	m := metrics{
		cmd: make([]commandMetrics, len(req.Cmds)),
	}
	m.req.request.count = 1
	m.req.request.size = len(req.Cmds)

	for i, cmd := range req.Cmds {
		c := &m.cmd[i]
		c.name = cmd.Cmd
		c.command.count = 1
		c.command.size = argsLen(cmd.Args)
	}

	res, err := t.rt.RoundTrip(req)

	if err != nil {
		observeResponse(t.engine, &m, err, start, time.Now())
	} else {
		if res.Args != nil {
			res.Args = &args{Args: res.Args, start: start, engine: t.engine, metrics: m}
		}
		if res.TxArgs != nil {
			res.TxArgs = &txArgs{TxArgs: res.TxArgs, start: start, engine: t.engine, metrics: m}
		}
	}

	return res, err
}

type args struct {
	redis.Args
	start  time.Time
	once   sync.Once
	engine *stats.Engine
	rsize  int
	metrics
}

func (a *args) Close() error {
	err := a.Args.Close()
	a.once.Do(func() {
		for i := range a.cmd {
			a.cmd[i].command.rsize = a.rsize
		}
		observeResponse(a.engine, &a.metrics, err, a.start, time.Now())
	})
	return err
}

func (a *args) Next(dst interface{}) bool {
	ok := a.Args.Next(dst)
	if ok {
		a.rsize += valueLen(reflect.ValueOf(dst))
	}
	return ok
}

type txArgs struct {
	redis.TxArgs
	start  time.Time
	once   sync.Once
	engine *stats.Engine
	index  int
	metrics
}

func (a *txArgs) Next() redis.Args {
	sub := a.TxArgs.Next()
	if sub != nil {
		sub = &txSubArgs{Args: sub, start: a.start, cmd: &a.cmd[a.index]}
		a.index++
	}
	return sub
}

func (a *txArgs) Close() error {
	err := a.TxArgs.Close()
	a.once.Do(func() { observeResponse(a.engine, &a.metrics, err, a.start, time.Now()) })
	return err
}

type txSubArgs struct {
	redis.Args
	start time.Time
	once  sync.Once
	rsize int
	cmd   *commandMetrics
}

func (a *txSubArgs) Close() error {
	err := a.Args.Close()
	a.once.Do(func() {
		a.cmd.command.rsize = a.rsize
		a.cmd.command.rtt = time.Now().Sub(a.start)

		// Only protocol errors are reported here, other kinds of errors are
		// fatal and are reported on all commands.
		if e, ok := err.(*resp.Error); ok {
			a.cmd.error.count = 1
			a.cmd.error.operation = "command"
			a.cmd.error.errtype = e.Type()
		}
	})
	return err
}

func (a *txSubArgs) Next(dst interface{}) bool {
	ok := a.Args.Next(dst)
	if ok {
		a.rsize += valueLen(reflect.ValueOf(dst))
	}
	return ok
}

func observeResponse(eng *stats.Engine, m *metrics, err error, start time.Time, end time.Time) {
	rtt := end.Sub(start)

	for i := range m.cmd {
		if c := &m.cmd[i]; c.command.rtt == 0 {
			c.command.rtt = rtt
		}
	}

	if errCount, errOp, errType := errorInfo(err); errCount != 0 {
		for i := range m.cmd {
			if c := &m.cmd[i]; c.error.count == 0 {
				c.error.count = errCount
				c.error.operation = errOp
				c.error.errtype = errType
			}
		}
	}

	eng.ReportAt(start, &m.req)
	eng.ReportAt(start, &m.cmd)
}

func argsLen(args redis.Args) int {
	if args == nil {
		return 0
	}
	return args.Len()
}

func valueLen(v reflect.Value) int {
	if v.IsValid() {
		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			return v.Len()
		case reflect.Ptr:
			if !v.IsNil() {
				return valueLen(v.Elem())
			}
		default:
			return 1
		}
	}
	return 0
}

func errorInfo(err error) (cnt int, op string, typ string) {
	switch e := err.(type) {
	case nil:
	case *net.OpError:
		cnt, op, typ = 1, e.Op, "network"
	case *resp.Error:
		cnt, op, typ = 1, "command", e.Type()
	default:
		cnt = 1
	}
	return
}
