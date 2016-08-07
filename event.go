package stats

import "sync"

type Event struct {
	Type  string  `json:"type"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Tags  Tags    `json:"tags,omitempty"`
}

func MakeEvent(m Metric, v float64) Event {
	return Event{
		Type:  m.Type(),
		Name:  m.Name(),
		Tags:  m.Tags(),
		Value: v,
	}
}

type EventBackend struct {
	sync.RWMutex
	Events []Event
}

func (b *EventBackend) Close() error { return nil }

func (b *EventBackend) Set(m Metric, v float64) { b.call(m, v) }

func (b *EventBackend) Add(m Metric, v float64) { b.call(m, v) }

func (b *EventBackend) Observe(m Metric, v float64) { b.call(m, v) }

func (b *EventBackend) call(m Metric, v float64) {
	b.Lock()
	defer b.Unlock()
	b.Events = append(b.Events, MakeEvent(m, v))
}
