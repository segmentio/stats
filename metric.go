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

	Set(float64) error
}

type Counter interface {
	Metric

	Add(float64) error
}

type Histogram interface {
	Metric

	Observe(time.Duration) error
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
	set func(Metric, float64) error
}

func NewGauge(opts Opts, set func(Metric, float64) error) gauge {
	return gauge{metric: makeMetric(opts), set: set}
}

func (g gauge) Set(x float64) error { return g.set(g, x) }

type counter struct {
	metric
	add func(Metric, float64) error
}

func NewCounter(opts Opts, add func(Metric, float64) error) counter {
	return counter{metric: makeMetric(opts), add: add}
}

func (c counter) Add(x float64) error { return c.add(c, x) }

type histogram struct {
	metric
	obs func(Metric, time.Duration) error
}

func NewHistogram(opts Opts, obs func(Metric, time.Duration) error) histogram {
	return histogram{metric: makeMetric(opts), obs: obs}
}

func (h histogram) Observe(x time.Duration) error { return h.obs(h, x) }

func JoinMetricName(sep string, elems ...string) string {
	parts := make([]string, 0, len(elems))

	for _, elem := range elems {
		if len(elem) != 0 {
			parts = append(parts, elem)
		}
	}

	return strings.Join(parts, sep)
}
