package stats

import (
	"io"
	"time"
)

type Backend interface {
	io.Closer

	Set(Metric, float64)

	Add(Metric, float64)

	Observe(Metric, time.Duration)
}

type BackendFunc func(Event)

func (b BackendFunc) Close() error { return nil }

func (b BackendFunc) Set(m Metric, v float64) { b.call(m, v) }

func (b BackendFunc) Add(m Metric, v float64) { b.call(m, v) }

func (b BackendFunc) Observe(m Metric, v time.Duration) { b.call(m, v) }

func (b BackendFunc) call(m Metric, v interface{}) { b(MakeEvent(m, v)) }

func MultiBackend(backends ...Backend) Backend {
	return multiBackend(backends)
}

type multiBackend []Backend

func (b multiBackend) Close() (err error) {
	for _, x := range b {
		err = appendError(err, x.Close())
	}
	return
}

func (b multiBackend) Set(m Metric, v float64) {
	for _, x := range b {
		x.Set(m, v)
	}
}

func (b multiBackend) Add(m Metric, v float64) {
	for _, x := range b {
		x.Add(m, v)
	}
}

func (b multiBackend) Observe(m Metric, v time.Duration) {
	for _, x := range b {
		x.Observe(m, v)
	}
}
