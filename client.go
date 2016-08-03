package stats

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

type Client interface {
	io.Closer

	Gauge(Opts) Gauge

	Counter(Opts) Counter

	Histogram(Opts) Histogram
}

type Config struct {
	Output io.Writer
	Scope  string
	Tags   Tags
}

func NewClient(out io.Writer) Client {
	return NewClientWith(Config{
		Output: out,
	})
}

func NewClientWith(config Config) Client {
	return client{
		output: bufio.NewWriter(config.Output),
		scope:  config.Scope,
		tags:   config.Tags,
	}
}

type client struct {
	output *bufio.Writer
	scope  string
	tags   Tags
}

func (c client) Close() error {
	return c.output.Flush()
}

func (c client) Gauge(opts Opts) Gauge {
	return NewGauge(c.opts(opts), c.set)
}

func (c client) Counter(opts Opts) Counter {
	return NewCounter(c.opts(opts), c.add)
}

func (c client) Histogram(opts Opts) Histogram {
	return NewHistogram(c.opts(opts), c.observe)
}

func (c client) opts(opts Opts) Opts {
	if len(opts.Scope) == 0 {
		opts.Scope = c.scope
	}
	opts.Tags = append(opts.Tags, c.tags...)
	return opts
}

func (c client) set(m Metric, x float64) { c.send("gauge", m, x) }

func (c client) add(m Metric, x float64) { c.send("counter", m, x) }

func (c client) observe(m Metric, x time.Duration) { c.send("histogram", m, x.Seconds()) }

func (c client) send(t string, m Metric, v float64) {
	json.NewEncoder(c.output).Encode(struct {
		Type  string      `json:"type"`
		Name  string      `json:"name"`
		Help  string      `json:"help,omitempty"`
		Value interface{} `json:"value"`
		Tags  Tags        `json:"tags,omitempty"`
	}{
		Type:  t,
		Name:  m.Name(),
		Help:  m.Help(),
		Value: v,
		Tags:  m.Tags(),
	})
}
