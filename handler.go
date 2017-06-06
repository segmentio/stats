package stats

// Handler is an interface implemented by types that receive metrics and expose
// them to diverse platforms.
//
// Handlers are usually not used directly, instead they are hooked to an engine
// which provides a higher-level abstraction to generate and publish metrics.
type Handler interface {
	// HandleMetric is called to report metrics produced by the program.
	//
	// The handler does not have ownership of the metric object it receives, it
	// must not retain the object or any of its field.
	HandleMetric(*Metric)
}

// HandlerFunc makes it possible for simple functions to be used as metric
// handlers.
type HandlerFunc func(*Metric)

// HandleMetric calls f.
func (f HandlerFunc) HandleMetric(m *Metric) {
	f(m)
}

// Flusher is an interface that may be implemented by metric handlers that do
// buffering or caching of the metrics they receive.
type Flusher interface {
	// Flush is called when the object should flush out all data it has cached
	// or buffered internally.
	Flush()
}

// StripTags returns a decorated version of handler that filters out the given
// list of tag names from all handled metrics.
func StripTags(handler Handler, tags ...string) Handler {
	index := make(map[string]struct{}, len(tags))

	for _, tag := range tags {
		index[tag] = struct{}{}
	}

	return HandlerFunc(func(m *Metric) {
		i := 0

		for _, tag := range m.Tags {
			if _, strip := index[tag.Name]; !strip {
				m.Tags[i] = tag
				i++
			}
		}

		m.Tags = m.Tags[:i]
		handler.HandleMetric(m)
	})
}
