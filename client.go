package stats

import (
	"io"
	"os"
	"path/filepath"
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
	Sample  float64
	Now     func() time.Time
}

var (
	DefaultScope string
)

func NewClient(scope string, backend Backend, tags ...Tag) Client {
	return NewClientWith(Config{
		Backend: backend,
		Scope:   scope,
		Tags:    copyTags(tags),
	})
}

func NewClientWith(config Config) Client {
	return client{
		backend: config.Backend,
		scope:   config.Scope,
		tags:    config.Tags,
		sample:  config.Sample,
		now:     config.Now,
	}
}

type client struct {
	backend Backend
	scope   string
	tags    Tags
	sample  float64
	now     func() time.Time
}

func (c client) Close() error { return c.backend.Close() }

func (c client) Gauge(name string, tags ...Tag) Gauge {
	return NewGauge(c.opts(name, tags...))
}

func (c client) Counter(name string, tags ...Tag) Counter {
	return NewCounter(c.opts(name, tags...))
}

func (c client) Histogram(name string, tags ...Tag) Histogram {
	return NewHistogram(c.opts(name, tags...))
}

func (c client) Timer(name string, tags ...Tag) Timer {
	return NewTimer(c.opts(name, tags...))
}

func (c client) opts(name string, tags ...Tag) Opts {
	return Opts{
		Backend: c.backend,
		Scope:   c.scope,
		Name:    name,
		Sample:  c.sample,
		Tags:    concatTags(c.tags, copyTags(Tags(tags))),
		Now:     c.now,
	}
}

func init() {
	DefaultScope = defaultScope()
}

func defaultScope() (scope string) {
	if len(os.Args) != 0 {
		scope = filepath.Base(os.Args[0])
	}
	return
}
