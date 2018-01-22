package netstats

import (
	"context"
	"net"

	"github.com/segmentio/stats"
)

// Handler is an interface that can be implemented by types that serve network
// connections.
type Handler interface {
	ServeConn(ctx context.Context, conn net.Conn)
}

// NewHandler returns a Handler object that warps hdl and produces
// metrics on the default engine.
func NewHandler(hdl Handler) Handler {
	return NewHandlerWith(stats.DefaultEngine, hdl)
}

// NewHandlerWith returns a Handler object that warps hdl and produces
// metrics on eng.
func NewHandlerWith(eng *stats.Engine, hdl Handler) Handler {
	return &handler{
		handler: hdl,
		eng:     eng,
	}
}

type handler struct {
	handler Handler
	eng     *stats.Engine
}

func (h *handler) ServeConn(ctx context.Context, conn net.Conn) {
	h.handler.ServeConn(ctx, NewConnWith(h.eng, conn))
}
