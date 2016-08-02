package stats

type Handler interface {
	Handle(Metric, Value) error
}

type HandlerFunc func(Metric, Value) error

func (f HandlerFunc) Handle(m Metric, v Value) error { return f(m, v) }

type Tracker interface {
	Incr(Metric, Value)

	Decr(Metric, Value)

	Track(Metric, Value)
}

type TrackerFunc func(Metric, Value)

func (f TrackerFunc) Incr(m Metric, v Value) { f.Track(m, Incr(v)) }

func (f TrackerFunc) Decr(m Metric, v Value) { f.Track(m, Decr(v)) }

func (f TrackerFunc) Track(m Metric, v Value) { f(m, v) }

func NewTracker(handler Handler, onError func(Metric, Value, error)) Tracker {
	if onError == nil {
		panic("stats.NewTracker: error handler cannot be nil")
	}
	return TrackerFunc(func(metric Metric, value Value) {
		if err := handler.Handle(metric, value); err != nil {
			onError(metric, value, err)
		}
	})
}

func NewDispatcher(trackers ...Tracker) Tracker {
	return TrackerFunc(func(metric Metric, value Value) {
		for _, tracker := range trackers {
			tracker.Track(metric, value)
		}
	})
}
