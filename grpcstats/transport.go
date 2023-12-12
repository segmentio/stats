package grpcstats

import (
	"net/http"
	"time"

	"github.com/segmentio/stats/v4"
)

type transport {
	transport http.RoundTripper
	eng *stats.Engine
}

// NewTransport wraps t to produce metrics on the default engine for every request
// sent and every response received.
func NewTransport(t http.RoundTripper) http.RoundTripper {
	return NewTransportWith(stats.DefaultEngine, t)
}

// NewTransportWith wraps t to produce metrics on eng for every request sent and
// every response received.
func NewTransportWith(eng *stats.Engine, t http.RoundTripper) http.RoundTripper {
	return &transport{
		transport: t,
		eng:       eng,
	}
}