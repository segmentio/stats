package stats

import (
	"runtime"
	"sync"
	"time"
)

const (
	DefaultMaxPending    = 1000
	DefaultMetricTimeout = 1 * time.Minute
)

type EngineConfig struct {
	Prefix        string
	MaxPending    int
	MetricTimeout time.Duration
}

type Engine struct {
	opch chan<- metricOp
	mqch chan<- metricReq
	once sync.Once
}

var (
	DefaultEngine = NewDefaultEngine()
)

func NewDefaultEngine() *Engine {
	return NewEngine(EngineConfig{})
}

func NewEngine(config EngineConfig) *Engine {
	if config.MaxPending == 0 {
		config.MaxPending = DefaultMaxPending
	}

	if config.MetricTimeout == 0 {
		config.MetricTimeout = DefaultMetricTimeout
	}

	opch := make(chan metricOp, config.MaxPending)
	mqch := make(chan metricReq)

	eng := &Engine{
		opch: opch,
		mqch: mqch,
	}

	go runEngine(config.Prefix, opch, mqch, makeMetricStore(metricStoreConfig{
		timeout: config.MetricTimeout,
	}))

	runtime.SetFinalizer(eng, (*Engine).Close)
	return eng
}

func (eng *Engine) Counter(name string, tags ...Tag) Counter {
	return makeCounter(eng, name, copyTags(tags))
}

func (eng *Engine) Gauge(name string, tags ...Tag) Gauge {
	return makeGauge(eng, name, copyTags(tags))
}

func (eng *Engine) Close() error {
	eng.once.Do(eng.close)
	return nil
}

func (eng *Engine) close() {
	close(eng.opch)
	close(eng.mqch)
}

func (eng *Engine) State() []Metric {
	res := make(chan []Metric, 1)
	eng.mqch <- metricReq{res: res}
	return <-res
}

func (eng *Engine) push(op metricOp) {
	if eng == nil {
		eng = DefaultEngine
	}
	eng.opch <- op
}

func runEngine(prefix string, opch <-chan metricOp, mqch <-chan metricReq, store metricStore) {
	ticker := time.NewTicker(store.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case op, ok := <-opch:
			if !ok {
				return // done
			}

			if len(prefix) != 0 {
				op.name = prefix + "." + op.name
			}

			store.apply(op, time.Now())

		case mq, ok := <-mqch:
			if !ok {
				return // done
			}
			mq.res <- store.state()

		case now := <-ticker.C:
			store.deleteExpiredMetrics(now)
		}
	}
}
