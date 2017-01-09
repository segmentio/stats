package netstats

import (
	"context"
	"net"
	"testing"

	"github.com/segmentio/netx"
)

func TestHandler(t *testing.T) {
	conn := &testConn{}
	ok := false

	handler := NewHandler(nil, netx.HandlerFunc(func(ctx context.Context, conn net.Conn) {
		ok = true
	}))
	handler.ServeConn(context.Background(), conn)

	if !ok {
		t.Error("the connection handler wasn't called")
	}
}
