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

func (b backend) Set(m stats.Metric, v float64) error { return b.log(m, v) }

func (b backend) Add(m stats.Metric, v float64) error { return b.log(m, v) }

func (b backend) Observe(m stats.Metric, v time.Duration) error { return b.log(m, v.Seconds()) }

func (b backend) log(m stats.Metric, v float64) error {
	return b.WithFields(fields(m, v)).Info(m.Name())
}

func fields(m stats.Metric, v float64) log.Fields {
	return log.Fields{
		"metric": log.Fields{
			"name":  m.Name(),
			"type":  m.Type(),
			"help":  m.Help(),
			"tags":  tags(m.Tags()),
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
