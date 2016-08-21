package logrusstats

import (
	"github.com/Sirupsen/logrus"
	"github.com/segmentio/stats"
)

func NewBackend(logger *logrus.Logger) stats.Backend {
	return stats.BackendFunc(func(e stats.Event) {
		logger.WithFields(fields(e)).Info(e.Name)
	})
}

func fields(e stats.Event) logrus.Fields {
	return logrus.Fields{
		"metric": logrus.Fields{
			"name":   e.Name,
			"type":   e.Type,
			"value":  e.Value,
			"sample": e.Sample,
			"tags":   tags(e.Tags),
			"time":   e.Time,
		},
	}
}

func tags(tags stats.Tags) logrus.Fields {
	fields := make(logrus.Fields, len(tags))

	for _, tag := range tags {
		fields[tag.Name] = tag.Value
	}

	return fields
}
