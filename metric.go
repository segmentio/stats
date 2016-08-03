package stats

import (
	"strings"
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

type Opts struct {
	Scope string
	Name  string
	Unit  string
	Help  string
	Tags  Tags
}

type metric struct {
	name string
	help string
	tags Tags
}

func makeMetric(opts Opts) metric {
	return metric{
		name: JoinMetricName(".", opts.Scope, opts.Name, opts.Unit),
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

func NewGauge(opts Opts, set func(Metric, float64)) gauge {
	return gauge{metric: makeMetric(opts), set: set}
}

func (g gauge) Set(x float64) { g.set(g, x) }

type counter struct {
	metric
	add func(Metric, float64)
}

func NewCounter(opts Opts, add func(Metric, float64)) counter {
	return counter{metric: makeMetric(opts), add: add}
}

func (c counter) Add(x float64) { c.add(c, x) }

type histogram struct {
	metric
	obs func(Metric, time.Duration)
}

func NewHistogram(opts Opts, obs func(Metric, time.Duration)) histogram {
	return histogram{metric: makeMetric(opts), obs: obs}
}

func (h histogram) Observe(x time.Duration) { h.obs(h, x) }

func JoinMetricName(sep string, elems ...string) string {
	parts := make([]string, 0, len(elems))

	for _, elem := range elems {
		if len(elem) != 0 {
			parts = append(parts, elem)
		}
	}

	return strings.Join(parts, sep)
}
