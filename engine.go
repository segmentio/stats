package stats

import (
	"runtime"
	"sync"
	"time"
)

const (
	// DefaultMaxPending is the maximum number of in-flight metrics operations
	// on the default engine.
	DefaultMaxPending = 1000

	// DefaultMetricTimeout is the amount of time idle metrics are kept in the
	// default engine before being evicted.
	DefaultMetricTimeout = 1 * time.Minute
)

// EngineConfig carries the different configuration values that can be set when
// creating a new engine.
type EngineConfig struct {
	// Prefix is set on all metrics created for this engine.
	Prefix string

	// Tags is the extra list of tags that are set on all metrics of the engine.
	Tags []Tag

	// MaxPending is the maximum number of in-flight metrics operations on the
	// engine.
	MaxPending int

	// MetricTimeout is the amount of time idle metrics are kept in the engine
	// before being evicted.
	MetricTimeout time.Duration
}

// The Engine type receives metrics operations and stores metrocs states.
//
// The goal of this type is to maintain aggregated metric values between scraps
// from clients that expose those metrics to different collection systems.
type Engine struct {
	opch chan<- metricOp
	mqch chan<- metricReq
	once sync.Once
}

var (
	// DefaultEngine is the engine used by global metrics.
	DefaultEngine = NewDefaultEngine()
)

// NewDefaultEngine creates and returns an engine configured with default settings.
func NewDefaultEngine() *Engine {
	return NewEngine(EngineConfig{})
}

// NewEngine creates and returns an engine configured with config.
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

	go runEngine(config.Prefix, config.Tags, opch, mqch, makeMetricStore(metricStoreConfig{
		timeout: config.MetricTimeout,
	}))

	runtime.SetFinalizer(eng, (*Engine).Close)
	return eng
}

// Counter creates and returns a counter with name and tags that produces
// metrics on eng.
func (eng *Engine) Counter(name string, tags ...Tag) Counter {
	return makeCounter(eng, name, copyTags(tags))
}

// Gauge creates and returns a gauge with name and tags that produces
// metrics on eng.
func (eng *Engine) Gauge(name string, tags ...Tag) Gauge {
	return makeGauge(eng, name, copyTags(tags))
}

// Close stops eng and releases all its allocated resources. After calling this
// method every use of metrics created for this engine will panic.
func (eng *Engine) Close() error {
	eng.once.Do(eng.close)
	return nil
}

// State returns the current state of the engine as a list of metrics.
func (eng *Engine) State() []Metric {
	res := make(chan []Metric, 1)
	eng.mqch <- metricReq{res: res}
	return <-res
}

func (eng *Engine) close() {
	close(eng.opch)
	close(eng.mqch)
}

func (eng *Engine) push(op metricOp) {
	if eng == nil {
		eng = DefaultEngine
	}
	eng.opch <- op
}

func runEngine(prefix string, tags []Tag, opch <-chan metricOp, mqch <-chan metricReq, store metricStore) {
	ticker := time.NewTicker(store.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case op, ok := <-opch:
			if !ok {
				return // done
			}

			rekey := false

			if len(tags) != 0 {
				rekey = true
				op.tags = append(op.tags, tags...)
				sortTags(op.tags)
			}

			if len(prefix) != 0 {
				rekey = true
				op.name = prefix + op.name
			}

			if rekey {
				op.key = metricKey(op.name, op.tags)
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
