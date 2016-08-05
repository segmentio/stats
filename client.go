package stats

import (
	"io"
	"time"
)

type Client interface {
	io.Closer

	Gauge(name string, tags ...Tag) Gauge

	Counter(name string, tags ...Tag) Counter

	Histogram(name string, tags ...Tag) Histogram

	Timer(name string, tags ...Tag) Timer
}

type Config struct {
	Backend Backend
	Scope   string
	Tags    Tags
	Now     func() time.Time
}

func NewClient(scope string, backend Backend, tags ...Tag) Client {
	return NewClientWith(Config{
		Backend: backend,
		Scope:   scope,
		Tags:    tags,
	})
}

func NewClientWith(config Config) Client {
	return client{
		backend: config.Backend,
		scope:   config.Scope,
		tags:    config.Tags,
		now:     config.Now,
	}
}

type client struct {
	backend Backend
	scope   string
	tags    Tags
	now     func() time.Time
}

func (c client) Close() error { return c.backend.Close() }

func (c client) Gauge(name string, tags ...Tag) Gauge {
	return NewGauge(c.backend, c.opts(name, tags...))
}

func (c client) Counter(name string, tags ...Tag) Counter {
	return NewCounter(c.backend, c.opts(name, tags...))
}

func (c client) Histogram(name string, tags ...Tag) Histogram {
	return NewHistogram(c.backend, c.opts(name, tags...))
}

func (c client) Timer(name string, tags ...Tag) Timer {
	return NewTimerWith(c.now, c.backend, c.opts(name, tags...))
}

func (c client) opts(name string, tags ...Tag) Opts {
	return Opts{
		Scope: c.scope,
		Name:  name,
		Tags:  concatTags(c.tags, Tags(tags)),
	}
}
