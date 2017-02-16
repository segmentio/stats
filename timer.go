package stats

import "time"

// A Timer is a special case for a histogram that reports durations.
type Timer struct {
	eng  *Engine // the engine to produce metrics on
	name string  // the name of the timer
	tags []Tag   // the tags set on the timer
}

// NewTimer creates and returns a new timer producing a metric with name and
// tags on eng.
func NewTimer(eng *Engine, name string, tags ...Tag) *Timer {
	return &Timer{
		eng:  eng,
		name: name,
		tags: copyTags(tags),
	}
}

// Name returns the name of the timer.
func (t *Timer) Name() string {
	return t.name
}

// Tags returns the list of tags set on the timer.
//
// The method returns a reference to the timer's internal tag slice, it does
// not make a copy. It's expected that the program will treat this value as a
// read-only list and won't modify its content.
func (t *Timer) Tags() []Tag {
	return t.tags
}

// Clone returns a copy of the timer, potentially setting tags on the returned
// object.
func (t *Timer) Clone(tags ...Tag) *Timer {
	return &Timer{
		eng:  t.eng,
		name: t.name,
		tags: concatTags(t.tags, tags),
	}
}

// Start the timer, returning a clock object that should be used to publish the
// timer metrics.
func (t *Timer) Start() *Clock {
	return t.StartAt(time.Now())
}

// StartAt the timer with a predefined start time, returning a clock object that
// should be used to publish the timer metrics.
func (t *Timer) StartAt(now time.Time) *Clock {
	var tags []Tag

	if len(t.tags) != 0 {
		tags = make([]Tag, len(t.tags), len(t.tags)+1)
		copy(tags, t.tags)
	}

	return &Clock{
		metric: Histogram{
			eng:  t.eng,
			name: t.name,
			tags: tags,
		},
		last: now,
	}
}
