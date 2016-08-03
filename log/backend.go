package log_stats

import (
	"log"
	"time"

	"github.com/segmentio/stats"
)

func NewBackend(logger *log.Logger) stats.Backend {
	return backend{logger}
}

type backend struct {
	*log.Logger
}

func (b backend) Close() error { return nil }

func (b backend) Set(m stats.Metric, v float64) { b.log("gauge", m, v) }

func (b backend) Add(m stats.Metric, v float64) { b.log("counter", m, v) }

func (b backend) Observe(m stats.Metric, v time.Duration) { b.log("histogram", m, v) }

func (b backend) log(t string, m stats.Metric, v interface{}) {
	b.Printf("%s %s [%v] %v (%s)\n", t, m.Name(), m.Tags(), v, m.Help())
}
