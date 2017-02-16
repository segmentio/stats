package stats

import "time"

// The Clock type can be used to report statistics on durations.
//
// Clocks are useful to measure the duration taken by sequential execution steps
// and therefore aren't safe to be used concurrently by multiple goroutines.
type Clock struct {
	metric Histogram
	last   time.Time
}

// Name returns the name of the clock.
func (c *Clock) Name() string {
	return c.metric.Name()
}

// Tags returns the list of tags set on the clock.
//
// The method returns a reference to the clock's internal tag slice, it does
// not make a copy. It's expected that the program will treat this value as a
// read-only list and won't modify its content.
func (c *Clock) Tags() []Tag {
	return c.metric.Tags()
}

// Clone returns a copy of the clock, potentially setting tags on the returned
// object.
func (c *Clock) Clone(tags ...Tag) *Clock {
	return &Clock{
		metric: *c.metric.Clone(tags...),
		last:   c.last,
	}
}

// Stamp reports the time difference between now and the last time the method
// was called (or since the clock was created).
//
// The metric produced by this method call will have a "stamp" tag set to name.
func (c *Clock) Stamp(name string) {
	c.StampAt(name, time.Now())
}

// StampAt reports the time difference between now and the last time the method
// was called (or since the clock was created).
//
// The metric produced by this method call will have a "stamp" tag set to name.
func (c *Clock) StampAt(name string, now time.Time) {
	c.observe(name, now)
}

// Stop reports the time difference between now and the last time the Stamp
// method was called (or since the clock was created).
//
// The metric produced by this method call will have a "stamp" tag set to
// "total".
func (c *Clock) Stop() {
	c.StopAt(time.Now())
}

// StopAt reports the time difference between now and the last time the Stamp
// method was called (or since the clock was created).
//
// The metric produced by this method call will have a "stamp" tag set to
// "total".
func (c *Clock) StopAt(now time.Time) {
	c.observe("total", now)
}

func (c *Clock) observe(stamp string, now time.Time) {
	h := c.metric
	h.tags = append(h.tags, Tag{"stamp", stamp})
	h.Observe(now.Sub(c.last).Seconds())
	c.last = now
}
