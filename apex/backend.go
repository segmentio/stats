package apex_stats

import (
	"time"

	"github.com/apex/log"
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

func (b backend) Observe(m stats.Metric, v time.Duration) { b.log("histogram", m, v.Seconds()) }

func (b backend) log(t string, m stats.Metric, v float64) {
	b.WithFields(fields(t, m, v)).Info(m.Name())
}

func fields(t string, m stats.Metric, v float64) log.Fields {
	return log.Fields{
		"metric": log.Fields{
			"name":  m.Name(),
			"help":  m.Help(),
			"tags":  tags(m.Tags()),
			"type":  t,
			"value": v,
		},
	}
}

func tags(tags stats.Tags) log.Fields {
	fields := make(log.Fields, len(tags))

	for _, tag := range tags {
		fields[tag.Name] = tag.Value
	}

	return fields
}
