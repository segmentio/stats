package stats

import "sync"

// A Gauge represent a metric that reports a single value.
type Gauge struct {
	mutex sync.Mutex
	value float64 // current value of the gauge
	eng   *Engine // the engine to produce metrics on
	name  string  // the name of the gauge
	tags  []Tag   // the tags set on the gauge
}

// Name returns the name of the gauge.
func (g *Gauge) Name() string {
	return g.name
}

// Tags returns the list of tags set on the gauge.
//
// The method returns a reference to the gauge's internal tag slice, it does
// not make a copy. It's expected that the program will treat this value as a
// read-only list and won't modify its content.
func (g *Gauge) Tags() []Tag {
	return g.tags
}

// Value returns the current value of the gauge.
func (g *Gauge) Value() float64 {
	return g.value
}

// Clone returns a copy of the gauge, potentially setting tags on the returned
// object.
//
// The internal value of the returned gauge is set to zero.
func (g *Gauge) Clone(tags ...Tag) *Gauge {
	return &Gauge{
		eng:  g.eng,
		name: g.name,
		tags: concatTags(g.tags, tags),
	}
}

// Incr increments the gauge by a value of 1.
func (g *Gauge) Incr() {
	g.Add(1)
}

// Decr decrements the gauge by a value of 1.
func (g *Gauge) Decr() {
	g.Add(-1)
}

// Add adds a value to the gauge.
func (g *Gauge) Add(value float64) {
	g.mutex.Lock()
	g.value += value
	g.eng.Set(g.name, g.value, g.tags...)
	g.mutex.Unlock()
}

// Set sets the gauge to value.
func (g *Gauge) Set(value float64) {
	g.mutex.Lock()
	g.value = value
	g.eng.Set(g.name, value, g.tags...)
	g.mutex.Unlock()
}
