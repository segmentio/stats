package stats

import (
	"io"
	"math/rand"
	"time"
)

type Client interface {
	io.Closer

	Gauge(opts Opts) Gauge

	Counter(opts Opts) Counter

	Histogram(opts Opts) Histogram

	Timer(now time.Time, opts Opts) Timer
}

type Config struct {
	Backend Backend
	Scope   string
	Tags    Tags
	Rand    func() float64
}

func NewClient(scope string, backend Backend, tags ...Tag) Client {
	return NewClientWith(Config{
		Backend: backend,
		Scope:   scope,
		Tags:    tags,
	})
}

func NewClientWith(config Config) Client {
	if config.Rand == nil {
		config.Rand = rand.Float64
	}
	return client{
		backend: config.Backend,
		scope:   config.Scope,
		tags:    config.Tags.Copy(),
	}
}

type client struct {
	backend Backend
	scope   string
	tags    Tags
	rand    func() float64
}

func (c client) Close() error {
	return c.backend.Close()
}

func (c client) Gauge(opts Opts) Gauge {
	return NewGauge(c.opts(opts), c.backend.Set)
}

func (c client) Counter(opts Opts) Counter {
	return NewCounter(c.opts(opts), c.backend.Add)
}

func (c client) Histogram(opts Opts) Histogram {
	return NewHistogram(c.opts(opts), c.backend.Observe)
}

func (c client) Timer(now time.Time, opts Opts) Timer {
	return NewTimer(now, c.opts(opts), c.backend.Observe)
}

func (c client) opts(opts Opts) Opts {
	if len(opts.Scope) == 0 {
		opts.Scope = c.scope
	}

	n1 := len(c.tags)
	n2 := len(opts.Tags)

	tags := make(Tags, n1+n2)
	copy(tags, c.tags)
	copy(tags[n1:], opts.Tags)
	opts.Tags = tags

	if opts.Rand == nil {
		opts.Rand = c.rand
	}

	return opts
}
