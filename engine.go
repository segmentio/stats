package stats

import (
	"os"
	"path/filepath"
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
	DefaultMetricTimeout = 10 * time.Second
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
//
// Most applications don't need to create a stats engine and can simply use the
// default one which is implicitly used by metrics when no engine is specified.
type Engine struct {
	// error channel used to report dropped metrics.
	errch chan<- struct{}

	// operation and query channels used to communicate with the engine.
	opch  chan<- metricOp
	reqch chan<- metricReq

	// synchronization primitive to make Close idempotent.
	once sync.Once
}

var (
	// DefaultEngine is the engine used by global metrics.
	DefaultEngine = NewDefaultEngine()
)

// Incr increments by one the metric identified by name and tags, a new counter
// is created in the default engine if none existed.
func Incr(name string, tags ...Tag) {
	C(name, tags...).Incr()
}

// Add adds value to the metric identified by name and tags, a new counter is
// created in the default engine if none existed.
func Add(name string, value float64, tags ...Tag) {
	C(name, tags...).Add(value)
}

// Set sets the value of the metric identified by name and tags, a new gauge is
// created in the default engine if none existed.
func Set(name string, value float64, tags ...Tag) {
	G(name, tags...).Set(value)
}

// Time returns a clock that produces metrics with name and tags and can be used
// to report durations.
func Time(name string, start time.Time, tags ...Tag) *Clock {
	return T(name, tags...).StartAt(start)
}

// NewDefaultEngine creates and returns an engine configured with default settings.
func NewDefaultEngine() *Engine {
	return NewEngine(EngineConfig{Prefix: progname()})
}

// NewEngine creates and returns an engine configured with config.
func NewEngine(config EngineConfig) *Engine {
	if config.MaxPending == 0 {
		config.MaxPending = DefaultMaxPending
	}

	if config.MetricTimeout == 0 {
		config.MetricTimeout = DefaultMetricTimeout
	}

	errch := make(chan struct{}, 1)
	opch := make(chan metricOp, config.MaxPending)
	reqch := make(chan metricReq)

	eng := &Engine{
		errch: errch,
		opch:  opch,
		reqch: reqch,
	}

	go runEngine(engine{
		errch:  errch,
		opch:   opch,
		reqch:  reqch,
		prefix: config.Prefix,
		tags:   config.Tags,
		store:  makeMetricStore(metricStoreConfig{timeout: config.MetricTimeout}),
	})

	runtime.SetFinalizer(eng, (*Engine).Close)
	return eng
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
	eng.reqch <- metricReq{res: res}
	return <-res
}

func (eng *Engine) close() {
	close(eng.opch)
	close(eng.reqch)
}

func (eng *Engine) push(op metricOp) {
	if eng == nil {
		eng = DefaultEngine
	}

	select {
	case eng.opch <- op:
		return
	default:
		// Never block, we'd rather discard the metric than block the program.
	}

	select {
	case eng.errch <- struct{}{}:
	default:
		// Never block either, we may not report an accurate count of discarded
		// metrics but it's OK, the important part is giving a signal that some
		// metrics are getting discarded because of how loaded the metrics queue
		// is.
	}
}

// engine is the internal implementation of the Engine type, it carries the
// other end of the diverse communication channels and the local state on which
// the engine's goroutine works.
type engine struct {
	errch  <-chan struct{}
	opch   <-chan metricOp
	reqch  <-chan metricReq
	prefix string
	tags   []Tag
	store  metricStore
}

func runEngine(e engine) {
	ticker := time.NewTicker(e.store.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-e.errch:
			name := "stats.discarded"
			tags := e.tags

			if len(e.prefix) != 0 {
				name = e.prefix + name
			}

			e.store.apply(metricOp{
				typ:   CounterType,
				key:   metricKey(name, tags),
				name:  name,
				tags:  tags,
				value: 1,
			}, time.Now())

		case op, ok := <-e.opch:
			if !ok {
				return // done
			}
			rekey := false

			if len(e.tags) != 0 {
				rekey = true
				op.tags = append(op.tags, e.tags...)
				sortTags(op.tags)
			}

			if len(e.prefix) != 0 {
				rekey = true
				op.name = e.prefix + op.name
			}

			if rekey {
				op.key = metricKey(op.name, op.tags)
			}

			e.store.apply(op, time.Now())

		case req, ok := <-e.reqch:
			if !ok {
				return // done
			}
			req.res <- e.store.state()

		case now := <-ticker.C:
			e.store.deleteExpiredMetrics(now)
		}
	}
}

func progname() (name string) {
	if args := os.Args; len(args) != 0 {
		name = filepath.Base(args[0]) + "."
	}
	return
}
