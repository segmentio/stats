package stats

import (
	"strings"
	"sync"
	"time"
)

type Metric interface {
	Name() string

	Help() string

	Tags() Tags
}

type Gauge interface {
	Metric

	Set(float64)
}

type Counter interface {
	Metric

	Add(float64)
}

type Histogram interface {
	Metric

	Observe(time.Duration)
}

type Timer interface {
	Metric

	Lap(now time.Time, name string)

	Stop(now time.Time)
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
	name string
	help string
	tags Tags
}

func makeMetric(opts Opts) metric {
	return metric{
		name: JoinMetricName(opts.Scope, opts.Name, opts.Unit),
		help: opts.Help,
		tags: opts.Tags,
	}
}

func (m metric) Name() string { return m.name }
func (m metric) Help() string { return m.help }
func (m metric) Tags() Tags   { return m.tags }

type gauge struct {
	metric
	set func(Metric, float64)
}

func NewGauge(opts Opts, set func(Metric, float64)) Gauge {
	return gauge{metric: makeMetric(opts), set: set}
}

func (g gauge) Set(x float64) { g.set(g, x) }

type counter struct {
	metric
	add func(Metric, float64)
}

func NewCounter(opts Opts, add func(Metric, float64)) Counter {
	return counter{metric: makeMetric(opts), add: add}
}

func (c counter) Add(x float64) { c.add(c, x) }

type histogram struct {
	metric
	obs func(Metric, time.Duration)
}

func NewHistogram(opts Opts, obs func(Metric, time.Duration)) Histogram {
	return histogram{metric: makeMetric(opts), obs: obs}
}

func (h histogram) Observe(x time.Duration) { h.obs(h, x) }

type timer struct {
	metric
	start time.Time
	last  time.Time
	mtx   sync.Mutex
	obs   func(Metric, time.Duration)
}

func NewTimer(start time.Time, opts Opts, obs func(Metric, time.Duration)) Timer {
	return &timer{metric: makeMetric(opts), start: start, last: start, obs: obs}
}

func (t *timer) Lap(now time.Time, name string) {
	t.mtx.Lock()
	d := now.Sub(t.last)
	t.last = now
	t.mtx.Unlock()

	t.obs(t.histogram(name), d)
}

func (t *timer) Stop(now time.Time) {
	t.obs(t.histogram(""), now.Sub(t.start))
}

func (t *timer) histogram(name string) histogram {
	h := histogram{metric: t.metric, obs: t.obs}

	if len(name) != 0 {
		count := len(h.tags)
		tags := make(Tags, count+1)
		copy(tags, h.tags)
		tags[count] = Tag{
			Name:  "lap",
			Value: name,
		}
		h.tags = tags
	}

	return h
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
