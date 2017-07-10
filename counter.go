package stats

// A Counter represent a metric that is monotonically increasing.
type Counter struct {
	value f64     // current value of the counter
	eng   *Engine // the engine to produce metrics on
	name  string  // the name of the counter
	tags  []Tag   // the tags set on the counter
}

// Name returns the name of the counter.
func (c *Counter) Name() string {
	return c.name
}

// Tags returns the list of tags set on the counter.
//
// The method returns a reference to the counter's internal tag slice, it does
// not make a copy. It's expected that the program will treat this value as a
// read-only list and won't modify its content.
func (c *Counter) Tags() []Tag {
	return c.tags
}

// Value returns the current value of the counter.
func (c *Counter) Value() float64 {
	return c.value.float()
}

// WithTags returns a copy of the counter, potentially setting tags on the returned
// object.
//
// The internal value of the returned counter is set to zero.
func (c *Counter) WithTags(tags ...Tag) *Counter {
	return &Counter{
		eng:  c.eng,
		name: c.name,
		tags: concatTags(c.tags, tags),
	}
}

// Incr increments the counter by a value of 1.
func (c *Counter) Incr() {
	c.Add(1)
}

// Add adds a value to the counter.
//
// Note that most data collection systems expect counters to be monotonically
// increasing so the program should not call this method with negative values.
func (c *Counter) Add(value float64) {
	c.value.add(value)
	c.eng.Add(c.name, value, c.tags...)
}

// Set sets the value of the counter.
//
// Note that most data collection systems expect counters to be monotonically
// increasing. Calling Set may break this contract, it is the responsibility of
// the application to make sure it's not lowering the counter value.
//
// This method is useful for reporting values of counters that aren't managed
// by the application itself, like CPU ticks for example.
func (c *Counter) Set(value float64) {
	_, value = c.value.set(value)
	if value < 0 {
		value = -value
	}
	c.eng.Add(c.name, value, c.tags...)
}
