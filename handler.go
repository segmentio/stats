package stats

import (
	statsv5 "github.com/segmentio/stats/v5"
)

// Handler behaves like [stats/v5.Handler].
type Handler = statsv5.Handler

// Flusher behaves like [stats/v5.Flusher].
type Flusher = statsv5.Flusher

// HandlerFunc behaves like [stats/v5.HandlerFunc].
type HandlerFunc = statsv5.HandlerFunc

// MultiHandler behaves like [stats/v5.MultiHandler].
func MultiHandler(handlers ...Handler) Handler {
	return statsv5.MultiHandler(handlers...)
}

// FilteredHandler behaves like [stats/v5.FilteredHandler].
func FilteredHandler(h Handler, filter func([]Measure) []Measure) Handler {
	return statsv5.FilteredHandler(h, filter)
}

// Discard behaves like [stats/v5.Discard].
var Discard = statsv5.Discard
