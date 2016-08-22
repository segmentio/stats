package stats

import (
	"io"
	"time"
)

type Backend interface {
	io.Closer

	Set(Metric, float64, time.Time)

	Add(Metric, float64, time.Time)

	Observe(Metric, float64, time.Time)
}

type BackendFunc func(Event)

func (b BackendFunc) Close() error { return nil }

func (b BackendFunc) Set(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b BackendFunc) Add(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b BackendFunc) Observe(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b BackendFunc) call(m Metric, v float64, t time.Time) { b(MakeEvent(m, v, t)) }

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

func (b multiBackend) Set(m Metric, v float64, t time.Time) {
	for _, x := range b {
		x.Set(m, v, t)
	}
}

func (b multiBackend) Add(m Metric, v float64, t time.Time) {
	for _, x := range b {
		x.Add(m, v, t)
	}
}

func (b multiBackend) Observe(m Metric, v float64, t time.Time) {
	for _, x := range b {
		x.Observe(m, v, t)
	}
}

type discardBackend struct{}

func (_ discardBackend) Close() error                             { return nil }
func (_ discardBackend) Set(_ Metric, _ float64, _ time.Time)     {}
func (_ discardBackend) Add(_ Metric, _ float64, _ time.Time)     {}
func (_ discardBackend) Observe(_ Metric, _ float64, _ time.Time) {}

var (
	Discard Backend = discardBackend{}
)
