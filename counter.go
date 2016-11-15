package stats

// Counter is an immutable data strcture that can be used to represent metrics
// that accumulate values.
type Counter struct {
	eng  *Engine // the parent engine
	key  string  // cached metric key
	name string  // the name of the counter
	tags []Tag   // the tags set on the counter
}

// C returns a new counter that produces metrics on the default engine.
func C(name string, tags ...Tag) Counter {
	return makeCounter(nil, name, copyTags(tags))
}

// Name returns the name of the counter.
func (c Counter) Name() string {
	return c.name
}

// Tags returns the list of tags set on the counter.
//
// The returned slice is a copy of the internal slice maintained by the counter,
// the program owns it and can safely modify it without affecting the counter.
func (c Counter) Tags() []Tag {
	return copyTags(c.tags)
}

// Clone returns a copy of the counter, potentially setting tags on the returned
// object.
func (c Counter) Clone(tags ...Tag) Counter {
	return makeCounter(c.eng, c.name, append(copyTags(tags), c.tags...))
}

// Incr increments the counter by a value of 1.
func (c Counter) Incr() {
	c.Add(1)
}

// Add adds a value to the counter.
func (c Counter) Add(value float64) {
	c.eng.push(metricOp{
		typ:   CounterType,
		key:   c.key,
		name:  c.name,
		tags:  c.tags,
		value: value,
		apply: metricOpAdd,
	})
}

func makeCounter(eng *Engine, name string, tags []Tag) Counter {
	sortTags(tags)
	return Counter{
		eng:  eng,
		key:  metricKey(name, tags),
		name: name,
		tags: tags,
	}
}
