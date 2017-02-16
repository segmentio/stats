package stats

// A Histogram represent a metric that reports a distribution of observed
// values.
type Histogram struct {
	eng  *Engine // the engine to produce metrics on
	name string  // the name of the counter
	tags []Tag   // the tags set on the counter
}

// Name returns the name of the histogram.
func (h *Histogram) Name() string {
	return h.name
}

// Tags returns the list of tags set on the histogram.
//
// The method returns a reference to the histogram's internal tag slice, it does
// not make a copy. It's expected that the program will treat this value as a
// read-only list and won't modify its content.
func (h *Histogram) Tags() []Tag {
	return h.tags
}

// Clone returns a copy of the histogram, potentially setting tags on the
// returned object.
func (h *Histogram) Clone(tags ...Tag) *Histogram {
	return &Histogram{
		eng:  h.eng,
		name: h.name,
		tags: concatTags(h.tags, tags),
	}
}

// Observe reports a value observed by the histogram.
func (h *Histogram) Observe(value float64) {
	h.eng.Observe(h.name, value, h.tags...)
}
