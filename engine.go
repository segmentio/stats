package stats

import (
	"time"

	statsv5 "github.com/segmentio/stats/v5"
)

// Engine behaves like [stats/v5.Engine].
type Engine = statsv5.Engine

// NewEngine behaves like [stats/v5.NewEngine].
func NewEngine(prefix string, handler Handler, tags ...Tag) *Engine {
	return statsv5.NewEngine(prefix, handler, tags...)
}

// DefaultEngine behaves like [stats/v5.DefaultEngine].
var DefaultEngine = statsv5.DefaultEngine

// Register behaves like [stats/v5.Register].
func Register(handler Handler) {
	statsv5.Register(handler)
}

// Flush behaves like [stats/v5.Flush].
func Flush() {
	statsv5.Flush()
}

// WithPrefix behaves like [stats/v5.WithPrefix].
func WithPrefix(prefix string, tags ...Tag) *Engine {
	return statsv5.WithPrefix(prefix, tags...)
}

// WithTags behaves like [stats/v5.WithTags].
func WithTags(tags ...Tag) *Engine {
	return statsv5.WithTags(tags...)
}

// Incr behaves like [stats/v5.Incr].
func Incr(name string, tags ...Tag) {
	statsv5.Incr(name, tags...)
}

// IncrAt behaves like [stats/v5.IncrAt].
func IncrAt(time time.Time, name string, tags ...Tag) {
	statsv5.IncrAt(time, name, tags...)
}

// Add behaves like [stats/v5.Add].
func Add(name string, value interface{}, tags ...Tag) {
	statsv5.Add(name, value, tags...)
}

// AddAt behaves like [stats/v5.AddAt].
func AddAt(time time.Time, name string, value interface{}, tags ...Tag) {
	statsv5.AddAt(time, name, value, tags...)
}

// Set behaves like [stats/v5.Set].
func Set(name string, value interface{}, tags ...Tag) {
	statsv5.Set(name, value, tags...)
}

// SetAt behaves like [stats/v5.SetAt].
func SetAt(time time.Time, name string, value interface{}, tags ...Tag) {
	statsv5.SetAt(time, name, value, tags...)
}

// Observe behaves like [stats/v5.Observe].
func Observe(name string, value interface{}, tags ...Tag) {
	statsv5.Observe(name, value, tags...)
}

// ObserveAt behaves like [stats/v5.ObserveAt].
func ObserveAt(time time.Time, name string, value interface{}, tags ...Tag) {
	statsv5.ObserveAt(time, name, value, tags...)
}

// Report behaves like [stats/v5.Report].
func Report(metrics interface{}, tags ...Tag) {
	statsv5.Report(metrics, tags...)
}

// ReportAt behaves like [stats/v5.ReportAt].
func ReportAt(time time.Time, metrics interface{}, tags ...Tag) {
	statsv5.ReportAt(time, metrics, tags...)
}
