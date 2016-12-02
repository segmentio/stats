package stats

import "time"

// Timer is an immutable data strcture that can be used to represent metrics
// that accumulate values.
type Timer struct {
	eng  *Engine // the parent engine
	key  string  // cached metric key
	name string  // the name of the timer
	tags []Tag   // the tags set on the timer
}

// T returns a new timer that produces metrics on the default engine.
func T(name string, tags ...Tag) Timer {
	return MakeTimer(nil, name, tags...)
}

// MakeTimer returns a new timer that produces metrics on the given engine.
func MakeTimer(engine *Engine, name string, tags ...Tag) Timer {
	return makeTimer(engine, name, copyTags(tags))
}

// Name returns the name of the timer.
func (t Timer) Name() string {
	return t.name
}

// Tags returns the list of tags set on the timer.
//
// The returned slice is a copy of the internal slice maintained by the timer,
// the program owns it and can safely modify it without affecting the timer.
func (t Timer) Tags() []Tag {
	return copyTags(t.tags)
}

// Clone returns a copy of the timer, potentially setting tags on the returned
// object.
func (t Timer) Clone(tags ...Tag) Timer {
	if len(tags) == 0 {
		return t
	}
	return makeTimer(t.eng, t.name, concatTags(t.tags, tags))
}

// Start the timer, returning a clock object that should be used to publish the
// timer metrics.
func (t Timer) Start() *Clock {
	return t.StartAt(time.Now())
}

// StartAt the timer with a predefined start time, returning a clock object that
// should be used to publish the timer metrics.
func (t Timer) StartAt(now time.Time) *Clock {
	return &Clock{
		metric: Histogram{
			eng:  t.eng,
			key:  t.key,
			name: t.name,
			tags: t.tags,
		},
		last: now,
	}
}

func makeTimer(eng *Engine, name string, tags []Tag) Timer {
	sortTags(tags)
	return Timer{
		eng:  eng,
		key:  MetricKey(name, tags),
		name: name,
		tags: tags,
	}
}
