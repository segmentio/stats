package stats

// Counter is an immutable data strcture that can be used to represent metrics
// that accumulate values.
type Counter struct {
	key  string // cached metric key
	name string // the name of the counter
	tags []Tag  // the tags set on the counter
	opch chan<- metric
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

// Clone returns a copy of the counter, potentially adding tags to the returned
// object.
func (c Counter) Clone(tags ...Tag) Counter {
	return makeCounter(c.name, sortTags(concatTags(c.tags, tags)), c.opch)
}

// Incr increments the counter by a value of 1, potentially adding tags to the
// operation.
func (c Counter) Incr(tags ...Tag) {
	c.Add(1, tags...)
}

// Add adds value to the counter, potentially adding tags to the operation.
func (c Counter) Add(value float64, tags ...Tag) {
	c.Clone(tags...).add(value)
}

func (c Counter) add(value float64) {
	c.opch <- metric{
		key:   c.key,
		name:  c.name,
		tags:  c.tags,
		value: value,
	}
}

func makeCounter(name string, tags []Tag, opch chan<- metric) Counter {
	return Counter{
		key:  metricKey(name, tags),
		name: name,
		tags: tags,
		opch: opch,
	}
}
