package netstats

import (
	"context"
	"net"

	"github.com/segmentio/netx"
	"github.com/segmentio/stats"
)

// NewHandler returns a netx.Handler object that wraps handler and produces
// metrics on eng.
//
// If eng is nil, the default engine is used instead.
func NewHandler(eng *stats.Engine, handler netx.Handler, tags ...stats.Tag) netx.Handler {
	return netx.HandlerFunc(func(ctx context.Context, conn net.Conn) {
		handler.ServeConn(ctx, NewConn(eng, conn, tags...))
	})
}
