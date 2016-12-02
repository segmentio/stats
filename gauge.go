package stats

import "time"

// Gauge is an immutable data structure that can be used to represent metrics
// with a value that can go up or down.
type Gauge struct {
	eng  *Engine // the parent engine
	key  string  // cached metric key
	name string  // the name of the gauge
	tags []Tag   // the tags set on the gauge
}

// G returns a new gauge that produces metrics on the default engine.
func G(name string, tags ...Tag) Gauge {
	return MakeGauge(nil, name, tags...)
}

// MakeGauge returns a new gauge that produces metrics on the given engine.
func MakeGauge(engine *Engine, name string, tags ...Tag) Gauge {
	return makeGauge(engine, name, copyTags(tags))
}

// Name returns the name of the gauge.
func (g Gauge) Name() string {
	return g.name
}

// Tags returns the list of tags set on the gauge.
//
// The returned slice is a copy of the internal slice maintained by the gauge,
// the program owns it and can safely modify it without affecting the gauge.
func (g Gauge) Tags() []Tag {
	return copyTags(g.tags)
}

// Clone returns a copy of the gauge, potentially setting tags on the returned
// object.
func (g Gauge) Clone(tags ...Tag) Gauge {
	if len(tags) == 0 {
		return g
	}
	return makeGauge(g.eng, g.name, concatTags(g.tags, tags))
}

// Incr increments the gauge by a value of 1.
func (g Gauge) Incr() {
	g.Add(1)
}

// Decr decrements the gauge by a value of 1.
func (g Gauge) Decr() {
	g.Add(-1)
}

// Add adds a value to the gauge.
func (g Gauge) Add(value float64) {
	g.eng.Add(GaugeType, g.key, g.name, g.tags, value, time.Now())
}

// Set sets the gauge to value.
func (g Gauge) Set(value float64) {
	g.eng.Set(GaugeType, g.key, g.name, g.tags, value, time.Now())
}

func makeGauge(eng *Engine, name string, tags []Tag) Gauge {
	sortTags(tags)
	return Gauge{
		eng:  eng,
		key:  MetricKey(name, tags),
		name: name,
		tags: tags,
	}
}
