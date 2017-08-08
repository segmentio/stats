package rpcstats

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/segmentio/rpc"
	"github.com/segmentio/stats"
	"github.com/segmentio/stats/statstest"
	"github.com/stretchr/testify/assert"
)

type math struct{}

func (m math) Add(ctx context.Context, nums []int) (sum int, err error) {
	for _, n := range nums {
		sum += n
	}
	return
}

func (m math) Error(ctx context.Context, nums []int) (sum int, err error) {
	return 0, errors.New("test")
}

func (m math) Panic(ctx context.Context, nums []int) (sum int, err error) {
	panic("test")
}

func setup(t *testing.T) (*statstest.Handler, *rpc.Client, func()) {
	h := &statstest.Handler{}
	e := stats.NewEngine("", h)

	srv := rpc.NewServer()
	srv.Register("Math", NewInvokerWith(e, rpc.NewService(math{})))

	server := httptest.NewServer(srv)
	client := rpc.NewClient(rpc.ClientConfig{
		UserAgent: "math-client",
		URL:       server.URL + "/rpc",
	})

	return h, client, func() {
		server.Close()
	}
}

func TestInvokerNormal(t *testing.T) {
	h, client, done := setup(t)
	defer done()

	var res int
	err := client.Call(context.Background(), "Math.Add", []int{2, 2}, &res)
	assert.NoError(t, err)
	assert.Equal(t, 4, res)

	assert.Equal(t, 2, len(h.Measures()))

	for _, m := range h.Measures() {
		if !strings.Contains(m.Name, "jsonrpc") {
			t.Errorf("expected metric %q to contain \"jsonrpc\" but did not", m.Name)
		}

		assert.Equal(t, []stats.Tag{
			{"function", "Add"},
			{"method", "Math.Add"},
			{"resource", "Math"},
			{"result", "success"},
		}, m.Tags)
	}
}

func TestInvokerError(t *testing.T) {
	h, client, done := setup(t)
	defer done()

	var res int
	err := client.Call(context.Background(), "Math.Error", []int{2, 2}, &res)
	assert.Error(t, err)

	assert.Equal(t, 2, len(h.Measures()))

	for _, m := range h.Measures() {
		if !strings.Contains(m.Name, "jsonrpc") {
			t.Errorf("expected metric %q to contain \"jsonrpc\" but did not", m.Name)
		}

		assert.Equal(t, []stats.Tag{
			{"function", "Error"},
			{"method", "Math.Error"},
			{"resource", "Math"},
			{"result", "error"},
		}, m.Tags)
	}
}

func TestInvokerPanic(t *testing.T) {
	h, client, done := setup(t)
	defer done()

	var res int
	err := client.Call(context.Background(), "Math.Panic", []int{2, 2}, &res)
	assert.Error(t, err)

	assert.Equal(t, 2, len(h.Measures()))

	for _, m := range h.Measures() {
		if !strings.Contains(m.Name, "jsonrpc") {
			t.Errorf("expected metric %q to contain \"jsonrpc\" but did not", m.Name)
		}

		assert.Equal(t, []stats.Tag{
			{"function", "Panic"},
			{"method", "Math.Panic"},
			{"resource", "Math"},
			{"result", "panic"},
		}, m.Tags)
	}
}
