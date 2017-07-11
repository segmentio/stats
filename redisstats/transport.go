package redisstats

import (
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
	t.engine.Incr("requests", stats.Tag{Name: "command", Value: req.Cmd})

	// Instrument wrapped method call
	startTime := time.Now()
	res, err := t.rt.RoundTrip(req)
	endTime := time.Now()

	switch e := err.(type) {
	case nil:
		t.engine.ObserveDuration("request.rtt.seconds", endTime.Sub(startTime),
			stats.Tag{Name: "command", Value: req.Cmd},
			stats.Tag{Name: "upstream", Value: req.Addr})
	case *resp.Error:
		t.engine.Incr("errors",
			stats.Tag{Name: "type", Value: "response"},
			stats.Tag{Name: "upstream", Value: req.Addr})
	case *net.OpError:
		t.engine.Incr("errors",
			stats.Tag{Name: "type", Value: "network"},
			stats.Tag{Name: "operation", Value: e.Op},
			stats.Tag{Name: "upstream", Value: req.Addr})
	default:
		t.engine.Incr("errors",
			stats.Tag{Name: "upstream", Value: req.Addr})
	}

	return res, err
}
