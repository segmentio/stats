package stats

import (
	"strings"
	"sync"
	"time"
)

type Metric interface {
	Name() string

	Help() string

	Type() string

	Tags() Tags
}

type Gauge interface {
	Metric

	Set(value float64, tags ...Tag)
}

type Counter interface {
	Metric

	Add(value float64, tags ...Tag)
}

type Histogram interface {
	Metric

	Observe(value time.Duration, tags ...Tag)
}

type Timer interface {
	Metric

	Lap(now time.Time, name string, tags ...Tag)

	Stop(now time.Time, tags ...Tag)
}

type Opts struct {
	Scope string
	Name  string
	Unit  string
	Help  string
	Tags  Tags
}

func MakeOpts(name string, help string, tags ...Tag) Opts {
	return Opts{
		Name: name,
		Help: help,
		Tags: Tags(tags),
	}
}

type metric struct {
	name    string
	help    string
	tags    Tags
	backend Backend
}

func makeMetric(backend Backend, opts Opts) metric {
	return metric{
		name:    JoinMetricName(opts.Scope, opts.Name, opts.Unit),
		help:    opts.Help,
		tags:    opts.Tags,
		backend: backend,
	}
}

func (m metric) Name() string { return m.name }

func (m metric) Help() string { return m.help }

func (m metric) Tags() Tags { return m.tags }

func (m metric) clone(tags ...Tag) metric {
	c := m
	c.tags = concatTags(c.tags, Tags(tags))
	return c
}

type gauge struct{ metric }

func NewGauge(backend Backend, opts Opts) Gauge {
	return gauge{makeMetric(backend, opts)}
}

func (g gauge) Type() string {
	return "gauge"
}

func (g gauge) Set(value float64, tags ...Tag) {
	g.backend.Set(gauge{g.clone(tags...)}, value)
}

type counter struct{ metric }

func NewCounter(backend Backend, opts Opts) Counter {
	return counter{makeMetric(backend, opts)}
}

func (c counter) Type() string {
	return "counter"
}

func (c counter) Add(value float64, tags ...Tag) {
	c.backend.Add(counter{c.clone(tags...)}, value)
}

type histogram struct{ metric }

func NewHistogram(backend Backend, opts Opts) Histogram {
	return histogram{makeMetric(backend, opts)}
}

func (h histogram) Type() string {
	return "histogram"
}

func (h histogram) Observe(value time.Duration, tags ...Tag) {
	h.backend.Observe(histogram{h.clone(tags...)}, value)
}

type timer struct {
	metric
	start time.Time
	last  time.Time
	mtx   sync.Mutex
}

func NewTimer(start time.Time, backend Backend, opts Opts) Timer {
	return &timer{metric: makeMetric(backend, opts), start: start, last: start}
}

func (t *timer) Type() string {
	return "timer"
}

func (t *timer) Lap(now time.Time, name string, tags ...Tag) {
	t.mtx.Lock()
	d := now.Sub(t.last)
	t.last = now
	t.mtx.Unlock()

	t.backend.Observe(t.histogram(name, tags...), d)
}

func (t *timer) Stop(now time.Time, tags ...Tag) {
	t.backend.Observe(t.histogram("", tags...), now.Sub(t.start))
}

func (t *timer) histogram(name string, tags ...Tag) histogram {
	if len(name) != 0 {
		tags = append(tags, Tag{"lap", name})
	}
	return histogram{t.clone(tags...)}
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
