package stats

import (
	"strings"
	"sync"
	"time"
)

type Metric interface {
	Name() string

	Type() string

	Tags() Tags
}

type Gauge interface {
	Metric

	Set(value float64, tags ...Tag)
}

type Counter interface {
	Metric

	Set(value float64, tags ...Tag)

	Add(value float64, tags ...Tag)
}

type Histogram interface {
	Metric

	Observe(value float64, tags ...Tag)
}

type Timer interface {
	Metric

	Start(tags ...Tag) Clock
}

type Clock interface {
	Stamp(name string, tags ...Tag)

	Stop(tags ...Tag)
}

type Opts struct {
	Backend Backend
	Scope   string
	Name    string
	Unit    string
	Tags    Tags
}

func MakeOpts(name string, tags ...Tag) Opts {
	return Opts{
		Name: name,
		Tags: Tags(tags),
	}
}

type metric struct {
	name    string
	tags    Tags
	backend Backend
}

func makeMetric(opts Opts) metric {
	return metric{
		name:    JoinMetricName(opts.Scope, opts.Name, opts.Unit),
		tags:    opts.Tags,
		backend: opts.Backend,
	}
}

func (m metric) Name() string { return m.name }

func (m metric) Tags() Tags { return m.tags }

func (m metric) clone(tags ...Tag) metric {
	c := m
	c.tags = concatTags(c.tags, copyTags(Tags(tags)))
	return c
}

type gauge struct{ metric }

func NewGauge(opts Opts) Gauge { return &gauge{makeMetric(opts)} }

func (g *gauge) Type() string { return "gauge" }

func (g *gauge) Set(value float64, tags ...Tag) {
	g.backend.Set(&gauge{g.clone(tags...)}, value)
}

type counter struct {
	sync.Mutex
	metric
	value float64
}

func NewCounter(opts Opts) Counter { return &counter{metric: makeMetric(opts)} }

func (c *counter) Type() string { return "counter" }

func (c *counter) Set(value float64, tags ...Tag) {
	c.Lock()
	defer c.Unlock()
	c.backend.Add(&counter{metric: c.clone(tags...), value: value}, value-c.value)
	c.value = value
}

func (c *counter) Add(value float64, tags ...Tag) {
	c.Lock()
	defer c.Unlock()
	c.value += value
	c.backend.Add(&counter{metric: c.clone(tags...), value: c.value}, value)
}

type histogram struct{ metric }

func NewHistogram(opts Opts) Histogram { return &histogram{makeMetric(opts)} }

func (h *histogram) Type() string { return "histogram" }

func (h *histogram) Observe(value float64, tags ...Tag) {
	h.backend.Observe(&histogram{h.clone(tags...)}, value)
}

type timer struct {
	metric
	now func() time.Time
}

func NewTimer(opts Opts) Timer {
	return NewTimerWith(nil, opts)
}

func NewTimerWith(now func() time.Time, opts Opts) Timer {
	if now == nil {
		now = time.Now
	}
	return &timer{metric: makeMetric(opts), now: now}
}

func (t *timer) Type() string { return "timer" }

func (t *timer) Start(tags ...Tag) Clock {
	now := t.now()
	return &clock{metric: t.clone(tags...), start: now, last: now, now: t.now}
}

type clock struct {
	metric
	start time.Time
	last  time.Time
	mtx   sync.Mutex
	now   func() time.Time
}

func (c *clock) Stamp(name string, tags ...Tag) {
	now := c.now()

	c.mtx.Lock()
	d := now.Sub(c.last)
	c.last = now
	c.mtx.Unlock()

	c.backend.Observe(c.histogram(name, tags...), d.Seconds())
}

func (c *clock) Stop(tags ...Tag) {
	c.backend.Observe(c.histogram("", tags...), c.now().Sub(c.start).Seconds())
}

func (c *clock) histogram(name string, tags ...Tag) *histogram {
	if len(name) != 0 {
		tags = append(tags, Tag{"stamp", name})
	}
	return &histogram{c.clone(tags...)}
}

func JoinMetricName(elems ...string) string {
	parts := make([]string, 0, len(elems))

	for _, elem := range elems {
		if len(elem) != 0 {
			parts = append(parts, elem)
		}
	}

	return strings.Join(parts, ".")
}
