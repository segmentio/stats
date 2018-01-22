package netstats

import (
	"context"
	"net"
	"testing"
)

type testHandler struct {
	ok bool
}

func (h *testHandler) ServeConn(ctx context.Context, conn net.Conn) {
	h.ok = true
}

func TestHandler(t *testing.T) {
	conn := &testConn{}
	test := &testHandler{}

	handler := NewHandler(test)
	handler.ServeConn(context.Background(), conn)

	if !test.ok {
		t.Error("the connection handler wasn't called")
	}
}
