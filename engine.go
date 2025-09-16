package stats

import (
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/segmentio/stats/v5/version"
)

// An Engine carries the context for producing metrics. It is configured by
// setting the exported fields or using the helper methods to create sub-engines
// that inherit the configuration of the base they were created from.
//
// The program must not modify the engine's handler, prefix, or tags after it
// starts using them. If changes need to be made new engines must be created by
// calls to WithPrefix or WithTags.
type Engine struct {
	// The measure handler that the engine forwards measures to.
	Handler Handler

	// A prefix set on all metric names produced by the engine.
	Prefix string

	// A list of tags set on all metrics produced by the engine.
	//
	// The list of tags has to be sorted. This is automatically managed by the
	// helper methods WithPrefix, WithTags and the NewEngine function. A program
	// that manipulates this field directly must respect this requirement.
	Tags []Tag

	// Indicates whether to allow duplicated tags from the tags list before sending.
	// This option is turned off by default, ensuring that duplicate tags are removed.
	// Turn it on if you need to send the same tag multiple times with different values,
	// which is a special use case.
	AllowDuplicateTags bool

	// This cache keeps track of the generated measure structures to avoid
	// rebuilding them every time a same measure type is seen by the engine.
	//
	// The cached values include the engine prefix in the measure names, which
	// is why the cache must be local to the engine.
	cache measureCache

	once sync.Once
}

// NewEngine creates and returns a new engine configured with prefix, handler,
// and tags.
func NewEngine(prefix string, handler Handler, tags ...Tag) *Engine {
	e := &Engine{
		Handler: handler,
		Prefix:  prefix,
		Tags:    SortTags(copyTags(tags)),
	}
	return e
}

// Register adds handler to e.
func (e *Engine) Register(handler Handler) {
	if e.Handler == Discard {
		e.Handler = handler
	} else {
		e.Handler = MultiHandler(e.Handler, handler)
	}
}

// Flush flushes eng's handler (if it implements the Flusher interface).
func (e *Engine) Flush() {
	flush(e.Handler)
}

// WithPrefix returns a copy of the engine with prefix appended to eng's current
// prefix and tags set to the merge of eng's current tags and those passed as
// argument. Both eng and the returned engine share the same handler.
func (e *Engine) WithPrefix(prefix string, tags ...Tag) *Engine {
	return &Engine{
		Handler: e.Handler,
		Prefix:  e.makeName(prefix),
		Tags:    mergeTags(e.Tags, tags),
	}
}

// SetPrefix returns a copy of the engine with prefix replacing the eng's current
// prefix and tags set to the merge of eng's current tags and those passed as
// argument. Both eng and the returned engine share the same handler.
func (e *Engine) SetPrefix(prefix string, tags ...Tag) *Engine {
	return &Engine{
		Handler: e.Handler,
		Prefix:  prefix,
		Tags:    mergeTags(e.Tags, tags),
	}
}

// WithTags returns a copy of the engine with tags set to the merge of eng's
// current tags and those passed as arguments. Both eng and the returned engine
// share the same handler.
func (e *Engine) WithTags(tags ...Tag) *Engine {
	return e.WithPrefix("", tags...)
}

// Incr increments by one the counter identified by name and tags.
func (e *Engine) Incr(name string, tags ...Tag) {
	e.Add(name, 1, tags...)
}

// IncrAt increments by one the counter identified by name and tags.
func (e *Engine) IncrAt(time time.Time, name string, tags ...Tag) {
	e.AddAt(time, name, 1, tags...)
}

// Add increments by value the counter identified by name and tags.
func (e *Engine) Add(name string, value interface{}, tags ...Tag) {
	e.measure(time.Now(), name, value, Counter, tags...)
}

// AddAt increments by value the counter identified by name and tags.
func (e *Engine) AddAt(t time.Time, name string, value interface{}, tags ...Tag) {
	e.measure(t, name, value, Counter, tags...)
}

// Set sets to value the gauge identified by name and tags.
func (e *Engine) Set(name string, value interface{}, tags ...Tag) {
	e.measure(time.Now(), name, value, Gauge, tags...)
}

// SetAt sets to value the gauge identified by name and tags.
func (e *Engine) SetAt(t time.Time, name string, value interface{}, tags ...Tag) {
	e.measure(t, name, value, Gauge, tags...)
}

// Observe reports value for the histogram identified by name and tags.
func (e *Engine) Observe(name string, value interface{}, tags ...Tag) {
	e.measure(time.Now(), name, value, Histogram, tags...)
}

// ObserveAt reports value for the histogram identified by name and tags.
func (e *Engine) ObserveAt(t time.Time, name string, value interface{}, tags ...Tag) {
	e.measure(t, name, value, Histogram, tags...)
}

// Clock returns a new clock identified by name and tags.
func (e *Engine) Clock(name string, tags ...Tag) *Clock {
	return e.ClockAt(name, time.Now(), tags...)
}

// ClockAt returns a new clock identified by name and tags with a specified
// start time.
func (e *Engine) ClockAt(name string, start time.Time, tags ...Tag) *Clock {
	cpy := make([]Tag, len(tags), len(tags)+1) // clock always appends a stamp.
	copy(cpy, tags)
	return &Clock{
		name:  name,
		first: start,
		last:  start,
		tags:  cpy,
		eng:   e,
	}
}

var truthyValues = map[string]bool{
	"true": true,
	"TRUE": true,
	"yes":  true,
	"1":    true,
	"on":   true,
}

var GoVersionReportingEnabled = !truthyValues[os.Getenv("STATS_DISABLE_GO_VERSION_REPORTING")]

func (e *Engine) reportVersionOnce(t time.Time) {
	if !GoVersionReportingEnabled {
		return
	}
	// We can't do this when we create the engine because it's possible to
	// configure it after creation time with e.g. the Register function. So
	// instead we try to do it at the moment you try to send your first metric.
	e.once.Do(func() {
		measures := []Measure{
			{
				Name: "stats_version",
				Fields: []Field{{
					Name:  "value",
					Value: intValue(1),
				}},
				Tags: []Tag{
					{"stats_version", version.Version},
				},
			},
		}
		// We don't want to report weird compiled Go versions like "devel" with
		// a commit SHA. Splitting on periods does not work as well for
		// filtering these
		if !version.DevelGoVersion() {
			measures = append(measures, Measure{
				Name: "go_version",
				Fields: []Field{{
					Name:  "value",
					Value: intValue(1),
				}},
				Tags: []Tag{
					{"go_version", version.GoVersion()},
				},
			})
		}
		e.Handler.HandleMeasures(t, measures...)
	})
}

func (e *Engine) measure(t time.Time, name string, value interface{}, ftype FieldType, tags ...Tag) {
	e.reportVersionOnce(t)
	e.measureOne(t, name, value, ftype, tags...)
}

func (e *Engine) measureOne(t time.Time, name string, value interface{}, ftype FieldType, tags ...Tag) {
	name, field := splitMeasureField(name)
	mp := measureArrayPool.Get().(*[1]Measure)

	m := &(*mp)[0]
	m.Name = e.makeName(name) // TODO: figure out how to optimize this
	m.Fields = append(m.Fields[:0], MakeField(field, value, ftype))
	m.Tags = append(m.Tags[:0], e.Tags...)
	m.Tags = append(m.Tags, tags...)

	if len(tags) != 0 && !e.AllowDuplicateTags && !TagsAreSorted(m.Tags) {
		SortTags(m.Tags)
	}

	e.Handler.HandleMeasures(t, (*mp)[:]...)

	for i := range m.Fields {
		m.Fields[i] = Field{}
	}

	for i := range m.Tags {
		m.Tags[i] = Tag{}
	}

	m.Name = ""
	measureArrayPool.Put(mp)
}

func (e *Engine) makeName(name string) string {
	return concat(e.Prefix, name)
}

var measureArrayPool = sync.Pool{
	New: func() interface{} { return new([1]Measure) },
}

// Report calls ReportAt with time.Now() as first argument.
func (e *Engine) Report(metrics interface{}, tags ...Tag) {
	e.ReportAt(time.Now(), metrics, tags...)
}

// ReportAt reports a set of metrics for a given time. The metrics must be of
// type struct, pointer to struct, or a slice or array to one of those. See
// MakeMeasures for details about how to make struct types exposing metrics.
func (e *Engine) ReportAt(t time.Time, metrics interface{}, tags ...Tag) {
	e.reportVersionOnce(t)
	var tb *tagsBuffer

	if len(tags) == 0 {
		// fast path for the common case where there are no dynamic tags
		tags = e.Tags
	} else {
		tb = tagsPool.Get().(*tagsBuffer)
		tb.append(tags...)
		tb.append(e.Tags...)
		if !e.AllowDuplicateTags {
			tb.sort()
		}
		tags = tb.tags
	}

	mb := measurePool.Get().(*measuresBuffer)
	mb.measures = appendMeasures(mb.measures[:0], &e.cache, e.Prefix, reflect.ValueOf(metrics), tags...)

	ms := mb.measures
	e.Handler.HandleMeasures(t, ms...)

	for i := range ms {
		ms[i].reset()
	}

	if tb != nil {
		tb.reset()
		tagsPool.Put(tb)
	}

	measurePool.Put(mb)
}

// DefaultEngine is the engine used by global helper functions.
var DefaultEngine = NewEngine(progname(), Discard)

// Register adds handler to the default engine.
func Register(handler Handler) {
	DefaultEngine.Register(handler)
}

// Flush flushes the default engine.
func Flush() {
	DefaultEngine.Flush()
}

// WithPrefix returns a copy of the engine with prefix appended to default
// engine's current prefix and tags set to the merge of engine's current tags
// and those passed as argument. Both the default engine and the returned engine
// share the same handler.
func WithPrefix(prefix string, tags ...Tag) *Engine {
	return DefaultEngine.WithPrefix(prefix, tags...)
}

// WithTags returns a copy of the engine with tags set to the merge of the
// default engine's current tags and those passed as arguments. Both the default
// engine and the returned engine share the same handler.
func WithTags(tags ...Tag) *Engine {
	return DefaultEngine.WithTags(tags...)
}

// Incr increments by one the counter identified by name and tags.
func Incr(name string, tags ...Tag) {
	DefaultEngine.Incr(name, tags...)
}

// IncrAt increments by one the counter identified by name and tags.
func IncrAt(time time.Time, name string, tags ...Tag) {
	DefaultEngine.IncrAt(time, name, tags...)
}

// Add increments by value the counter identified by name and tags.
func Add(name string, value interface{}, tags ...Tag) {
	DefaultEngine.Add(name, value, tags...)
}

// AddAt increments by value the counter identified by name and tags.
func AddAt(time time.Time, name string, value interface{}, tags ...Tag) {
	DefaultEngine.AddAt(time, name, value, tags...)
}

// Set sets to value the gauge identified by name and tags.
func Set(name string, value interface{}, tags ...Tag) {
	DefaultEngine.Set(name, value, tags...)
}

// SetAt sets to value the gauge identified by name and tags.
func SetAt(time time.Time, name string, value interface{}, tags ...Tag) {
	DefaultEngine.SetAt(time, name, value, tags...)
}

// Observe reports value for the histogram identified by name and tags.
func Observe(name string, value interface{}, tags ...Tag) {
	DefaultEngine.Observe(name, value, tags...)
}

// ObserveAt reports value for the histogram identified by name and tags.
func ObserveAt(time time.Time, name string, value interface{}, tags ...Tag) {
	DefaultEngine.ObserveAt(time, name, value, tags...)
}

// Report is a helper function that delegates to DefaultEngine.
func Report(metrics interface{}, tags ...Tag) {
	DefaultEngine.Report(metrics, tags...)
}

// ReportAt is a helper function that delegates to DefaultEngine.
func ReportAt(time time.Time, metrics interface{}, tags ...Tag) {
	DefaultEngine.ReportAt(time, metrics, tags...)
}

func progname() (name string) {
	if args := os.Args; len(args) != 0 {
		name = filepath.Base(args[0])
	}
	return
}
