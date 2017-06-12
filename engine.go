package stats

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// The Engine is the central system where metrics are reported and dispatched to
// handlers in charge of publishing them to various metrics platforms.
//
// Most applications don't need to create a stats engine and can simply use
// DefaultEngine, which is implicitly used by all top-level functions of the
// package.
type Engine struct {
	// immutable fields
	name string
	tags []Tag

	// mutable fields, synchronized on mutex
	mutex    sync.RWMutex
	handlers []Handler
	buckets  map[string][]float64
}

var (
	// DefaultEngine is the engine used by global metrics.
	//
	// Programs that need to change the default engine should do before creating
	// any metrics handlers or producers.
	DefaultEngine = NewEngine(progname())
)

// NewEngine creates and returns an engine with name and tags.
func NewEngine(name string, tags ...Tag) *Engine {
	return &Engine{
		name:    name,
		tags:    copyTags(tags),
		buckets: make(map[string][]float64),
	}
}

// Name returns the name of the engine.
func (eng *Engine) Name() string {
	return eng.name
}

// Tags returns a slice containing the tags set on the engine.
func (eng *Engine) Tags() []Tag {
	return eng.tags
}

// Handlers returns a slice containing the handlers currently set on the engine.
func (eng *Engine) Handlers() []Handler {
	eng.mutex.RLock()
	handlers := make([]Handler, len(eng.handlers))
	copy(handlers, eng.handlers)
	eng.mutex.RUnlock()
	return handlers
}

// Register adds handler to eng.
//
// To prevent any deadlock from happening this method should never be called
// from the handler's HandleMetric method.
func (eng *Engine) Register(handler Handler) {
	eng.mutex.Lock()
	eng.handlers = append(eng.handlers, handler)
	eng.mutex.Unlock()
}

// HistogramBuckets returns a map of metric names to buckets used to distribute
// hitogram values.
//
// The buckets are of list of upper limits used to group the observed values.
func (eng *Engine) HistogramBuckets() map[string][]float64 {
	eng.mutex.RLock()
	buckets := make(map[string][]float64, len(eng.buckets))

	for k, v := range eng.buckets {
		buckets[k] = copyBuckets(v)
	}

	eng.mutex.RUnlock()
	return buckets
}

// SetHistogramBuckets sets the buckets used for a histogram metric.
//
// Not all stats handler will respect the value distribution set with this
// method, refer to the documentation of the handler for more details.
func (eng *Engine) SetHistogramBuckets(name string, buckets ...float64) {
	if !sort.Float64sAreSorted(buckets) {
		panic("histogram buckets must be a sorted set of values")
	}
	buckets = copyBuckets(buckets)
	eng.mutex.Lock()
	eng.buckets[name] = buckets
	eng.mutex.Unlock()
}

// WithName creates a new engine which inherits the properties and handlers
// of eng and uses the given name.
func (eng *Engine) WithName(name string) *Engine {
	return &Engine{
		name:     name,
		tags:     eng.tags,
		handlers: eng.Handlers(),
		buckets:  eng.HistogramBuckets(),
	}
}

// WithTags creates a new engine which inherits the properties and handlers,
// adding the given tags to the returned engine.
func (eng *Engine) WithTags(tags ...Tag) *Engine {
	return &Engine{
		name:     eng.name,
		tags:     concatTags(eng.tags, tags),
		handlers: eng.Handlers(),
		buckets:  eng.HistogramBuckets(),
	}
}

// Flush flushes all handlers of eng that implement the Flusher interface.
func (eng *Engine) Flush() {
	eng.mutex.RLock()

	for _, h := range eng.handlers {
		if f, ok := h.(Flusher); ok {
			f.Flush()
		}
	}

	eng.mutex.RUnlock()
}

// Counter creates a new counter producing a metric with name and tag on eng.
func (eng *Engine) Counter(name string, tags ...Tag) *Counter {
	return &Counter{
		eng:  eng,
		name: name,
		tags: copyTags(tags),
	}
}

// Gauge creates a new gauge producing a metric with name and tag on eng.
func (eng *Engine) Gauge(name string, tags ...Tag) *Gauge {
	return &Gauge{
		eng:  eng,
		name: name,
		tags: copyTags(tags),
	}
}

// Histogram creates a new hitsogram producing a metric with name and tag on eng.
func (eng *Engine) Histogram(name string, tags ...Tag) *Histogram {
	return &Histogram{
		eng:  eng,
		name: name,
		tags: copyTags(tags),
	}
}

// Timer creates a new timer producing metrics with name and tag on eng.
func (eng *Engine) Timer(name string, tags ...Tag) *Timer {
	return &Timer{
		eng:  eng,
		name: name,
		tags: copyTags(tags),
	}
}

// Clock creates a new clock producing metrics with name and tags.
//
// The method is similar to `.Timer()`, except it will start the clock
// at `time.Now()`, this allows you to time functions nicely with `defer`:
//
// 			defer stats.Clock("myfunc").Stop()
//
// The method also allows you to time a series of operations, like so:
//
// 			c := stats.Clock("myfunc")
// 			...
// 			c.Stamp("download")
// 			...
// 			c.Stamp("process")
// 			c.Stop()
//
func (eng *Engine) Clock(name string, tags ...Tag) *Clock {
	return eng.Timer(name, tags...).Start()
}

// Incr increments by 1 the counter with name and tags on eng.
func (eng *Engine) Incr(name string, tags ...Tag) {
	eng.handle(CounterType, name, 1, tags, time.Time{})
}

// Add adds value to the counter with name and tags on eng.
func (eng *Engine) Add(name string, value float64, tags ...Tag) {
	eng.handle(CounterType, name, value, tags, time.Time{})
}

// Set sets the gauge with name and tags on eng to value.
func (eng *Engine) Set(name string, value float64, tags ...Tag) {
	eng.handle(GaugeType, name, value, tags, time.Time{})
}

// Observe reports a value on the histogram with name and tags on eng.
func (eng *Engine) Observe(name string, value float64, tags ...Tag) {
	eng.handle(HistogramType, name, value, tags, time.Time{})
}

// ObserveDuration reports a duration in seconds to the histogram with name and
// tags on eng.
func (eng *Engine) ObserveDuration(name string, value time.Duration, tags ...Tag) {
	eng.handle(HistogramType, name, value.Seconds(), tags, time.Time{})
}

func (eng *Engine) handle(typ MetricType, name string, value float64, tags []Tag, time time.Time) {
	var buckets []float64

	metric := metricPool.Get().(*Metric)
	eng.mutex.RLock()

	if typ == HistogramType {
		buckets = eng.buckets[name]
	}

	for _, handler := range eng.handlers {
		metric.Namespace = eng.name
		metric.Type = typ
		metric.Name = name
		metric.Value = value
		metric.Tags = append(metric.Tags[:0], eng.tags...)
		metric.Tags = append(metric.Tags, tags...)
		metric.Time = time
		metric.Buckets = buckets
		handler.HandleMetric(metric)
	}

	eng.mutex.RUnlock()
	metricPool.Put(metric)
}

// C returns a new counter that produces a metric with name and tags on the
// default engine.
func C(name string, tags ...Tag) *Counter {
	return DefaultEngine.Counter(name, tags...)
}

// G returns a new gauge that produces a metric with name and tags on the
// default engine.
func G(name string, tags ...Tag) *Gauge {
	return DefaultEngine.Gauge(name, tags...)
}

// H returns a new histogram that produces a metric with name and tags on the
// default engine.
func H(name string, tags ...Tag) *Histogram {
	return DefaultEngine.Histogram(name, tags...)
}

// T returns a new timer that produces a metric with name and tags on the
// default engine.
func T(name string, tags ...Tag) *Timer {
	return DefaultEngine.Timer(name, tags...)
}

// Incr increments by one the metric identified by name and tags, a new counter
// is created in the default engine if none existed.
func Incr(name string, tags ...Tag) {
	DefaultEngine.Incr(name, tags...)
}

// Add adds value to the metric identified by name and tags, a new counter is
// created in the default engine if none existed.
func Add(name string, value float64, tags ...Tag) {
	DefaultEngine.Add(name, value, tags...)
}

// Set sets the value of the metric identified by name and tags, a new gauge is
// created in the default engine if none existed.
func Set(name string, value float64, tags ...Tag) {
	DefaultEngine.Set(name, value, tags...)
}

// Observe reports a value for the metric identified by name and tags, a new
// histogram is created in the default engine if none existed.
func Observe(name string, value float64, tags ...Tag) {
	DefaultEngine.Observe(name, value, tags...)
}

// ObserveDuration reports a duration value of the metric identified by name
// and tags, a new timer is created in the default engine if none existed.
func ObserveDuration(name string, value time.Duration, tags ...Tag) {
	DefaultEngine.ObserveDuration(name, value, tags...)
}

// Time returns a clock that produces metrics with name and tags and can be used
// to report durations.
func Time(name string, start time.Time, tags ...Tag) *Clock {
	return DefaultEngine.Timer(name, tags...).StartAt(start)
}

// WithName creates a new engine which inherits the properties and handlers of
// the default handler and uses the given name.
func WithName(name string) *Engine {
	return DefaultEngine.WithName(name)
}

// WithTags creates a new engine which inherits the properties and handlers of
// the default handler and adds the given tags.
func WithTags(tags ...Tag) *Engine {
	return DefaultEngine.WithTags(tags...)
}

// Register adds handler to the default engine.
func Register(handler Handler) {
	DefaultEngine.Register(handler)
}

// Flush flushes all metrics on the default engine.
func Flush() {
	DefaultEngine.Flush()
}

func progname() (name string) {
	if args := os.Args; len(args) != 0 {
		name = filepath.Base(args[0])
	}
	return
}

func copyBuckets(buckets []float64) []float64 {
	return append(make([]float64, 0, len(buckets)), buckets...)
}
