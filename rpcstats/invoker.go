package rpcstats

import (
	"context"
	"strings"
	"time"

	"github.com/segmentio/rpc"
	"github.com/segmentio/stats"
)

type invoker struct {
	invoker rpc.Invoker
	eng     *stats.Engine
}

// NewInvoker wraps a RPC invoker to produce metrics for every RPC call.
func NewInvoker(i rpc.Invoker) rpc.Invoker {
	return NewInvokerWith(stats.DefaultEngine, i)
}

// NewInvoker wraps a RPC invoker to produce metrics on eng for every RPC call.
func NewInvokerWith(eng *stats.Engine, i rpc.Invoker) rpc.Invoker {
	return &invoker{
		invoker: i,
		eng:     eng,
	}
}

func (i *invoker) Invoke(ctx context.Context, req rpc.Request) (res interface{}, err error) {
	parts := strings.SplitN(req.Method, ".", 2)
	resource := parts[0]
	function := strings.Join(parts[1:], ".")

	var tags []stats.Tag
	tags = append(tags, stats.Tag{"method", req.Method})
	tags = append(tags, stats.Tag{"resource", resource})
	tags = append(tags, stats.Tag{"function", function})

	start := time.Now()

	defer func() {
		end := time.Now()

		if r := recover(); r != nil {
			tags = append(tags, stats.Tag{"result", "panic"})
			// We want to capture stats, but still propagate errors upstream.
			defer panic(r)
		} else if err != nil {
			tags = append(tags, stats.Tag{"result", "error"})
		} else {
			tags = append(tags, stats.Tag{"result", "success"})
		}

		i.eng.Incr("jsonrpc.call", tags...)
		i.eng.Observe("jsonrpc.duration", end.Sub(start), tags...)
	}()

	res, err = i.invoker.Invoke(ctx, req)
	return
}
