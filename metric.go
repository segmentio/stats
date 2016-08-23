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

	Sample() float64
}

type Gauge interface {
	Metric

	Set(value float64, tags ...Tag)

	SetAt(value float64, time time.Time, tags ...Tag)
}

type Counter interface {
	Metric

	Add(value float64, tags ...Tag)

	AddAt(value float64, time time.Time, tags ...Tag)
}

type Histogram interface {
	Metric

	Observe(value float64, tags ...Tag)

	ObserveAt(value float64, time time.Time, tags ...Tag)
}

type Timer interface {
	Metric

	Start(tags ...Tag) Clock

	StartAt(time time.Time, tags ...Tag) Clock
}

type Clock interface {
	Stamp(name string, tags ...Tag)

	StampAt(name string, time time.Time, tags ...Tag)

	Stop(tags ...Tag)

	StopAt(time time.Time, tags ...Tag)
}

type Opts struct {
	Backend Backend
	Scope   string
	Name    string
	Unit    string
	Tags    Tags
	Sample  float64
	Now     func() time.Time
}

func MakeOpts(name string, tags ...Tag) Opts { return Opts{Name: name, Tags: Tags(tags)} }

type metric struct {
	name    string
	tags    Tags
	backend Backend
	sample  float64
	now     func() time.Time
}

func makeMetric(opts Opts) metric {
	if opts.Now == nil {
		opts.Now = time.Now
	}

	if opts.Sample == 0 {
		opts.Sample = 1
	}

	return metric{
		name:    JoinMetricName(opts.Scope, opts.Name, opts.Unit),
		tags:    opts.Tags,
		backend: opts.Backend,
		sample:  opts.Sample,
		now:     opts.Now,
	}
}

func (m *metric) Name() string { return m.name }

func (m *metric) Tags() Tags { return m.tags }

func (m *metric) Sample() float64 { return m.sample }

func (m *metric) clone(tags ...Tag) metric {
	c := *m
	c.tags = concatTags(c.tags, copyTags(Tags(tags)))
	return c
}

type gauge struct{ metric }

func NewGauge(opts Opts) Gauge { return &gauge{makeMetric(opts)} }

func (g *gauge) Type() string { return "gauge" }

func (g *gauge) Set(value float64, tags ...Tag) { g.SetAt(value, g.now(), tags...) }

func (g *gauge) SetAt(value float64, time time.Time, tags ...Tag) {
	g.backend.Set(&gauge{g.clone(tags...)}, value, time)
}

type counter struct{ metric }

func NewCounter(opts Opts) Counter { return &counter{metric: makeMetric(opts)} }

func (c *counter) Type() string { return "counter" }

func (c *counter) Add(value float64, tags ...Tag) { c.AddAt(value, c.now(), tags...) }

func (c *counter) AddAt(value float64, time time.Time, tags ...Tag) {
	c.backend.Add(&counter{metric: c.clone(tags...)}, value, time)
}

type histogram struct{ metric }

func NewHistogram(opts Opts) Histogram { return &histogram{makeMetric(opts)} }

func (h *histogram) Type() string { return "histogram" }

func (h *histogram) Observe(value float64, tags ...Tag) { h.ObserveAt(value, h.now(), tags...) }

func (h *histogram) ObserveAt(value float64, time time.Time, tags ...Tag) {
	h.backend.Observe(&histogram{h.clone(tags...)}, value, time)
}

type timer struct{ metric }

func NewTimer(opts Opts) Timer {
	return &timer{metric: makeMetric(opts)}
}

func (t *timer) Type() string { return "timer" }

func (t *timer) Start(tags ...Tag) Clock { return t.StartAt(t.now(), tags...) }

func (t *timer) StartAt(time time.Time, tags ...Tag) Clock {
	return &clock{metric: t.clone(tags...), start: time, last: time}
}

type clock struct {
	sync.Mutex
	metric
	start time.Time
	last  time.Time
}

func (c *clock) Stamp(name string, tags ...Tag) { c.StampAt(name, c.now(), tags...) }

func (c *clock) StampAt(name string, time time.Time, tags ...Tag) {
	c.Lock()
	d := time.Sub(c.last)
	c.last = time
	c.Unlock()
	c.backend.Observe(c.histogram(name, tags...), d.Seconds(), time)
}

func (c *clock) Stop(tags ...Tag) { c.StopAt(c.now(), tags...) }

func (c *clock) StopAt(time time.Time, tags ...Tag) {
	c.backend.Observe(c.histogram("total", tags...), time.Sub(c.start).Seconds(), time)
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
