package apexstats

import (
	"github.com/apex/log"
	"github.com/segmentio/stats"
)

func NewBackend(logger log.Interface) stats.Backend {
	return stats.BackendFunc(func(e stats.Event) {
		logger.WithFields(fields(e)).Debug(e.Name)
	})
}

func fields(e stats.Event) log.Fields {
	return log.Fields{"metric": e}
}
