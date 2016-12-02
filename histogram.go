package stats

import "time"

// Histogram is an immutable data strcture that can be used to represent
// metrics that measure a distribution of values.
type Histogram struct {
	eng  *Engine // the parent engine
	key  string  // cached metric key
	name string  // the name of the histogram
	tags []Tag   // the tags set on the histogram
}

// H returns a new histogram that produces metrics on the default engine.
func H(name string, tags ...Tag) Histogram {
	return MakeHistogram(nil, name, tags...)
}

// MakeHistogram returns a new histogram that produces metrics on the given engine.
func MakeHistogram(engine *Engine, name string, tags ...Tag) Histogram {
	return makeHistogram(engine, name, copyTags(tags))
}

// Name returns the name of the histogram.
func (h Histogram) Name() string {
	return h.name
}

// Tags returns the list of tags set on the histogram.
//
// The returned slice is a copy of the internal slice maintained by the
// histogram, the program owns it and can safely modify it without affecting
// the histogram.
func (h Histogram) Tags() []Tag {
	return copyTags(h.tags)
}

// Clone returns a copy of the histogram, potentially setting tags on the
// returned object.
func (h Histogram) Clone(tags ...Tag) Histogram {
	if len(tags) == 0 {
		return h
	}
	return makeHistogram(h.eng, h.name, concatTags(h.tags, tags))
}

// Observe reports a value observed by the histogram.
func (h Histogram) Observe(value float64) {
	h.eng.Observe(HistogramType, h.key, h.name, h.tags, value, time.Now())
}

func makeHistogram(eng *Engine, name string, tags []Tag) Histogram {
	sortTags(tags)
	return Histogram{
		eng:  eng,
		key:  metricKey(name, tags),
		name: name,
		tags: tags,
	}
}
