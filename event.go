package stats

import "time"

type Event struct {
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	Help  string      `json:"help,omitempty"`
	Value interface{} `json:"value"`
	Tags  Tags        `json:"tags,omitempty"`
}

func MakeEvent(m Metric, v interface{}) Event {
	return Event{
		Type:  m.Type(),
		Name:  m.Name(),
		Help:  m.Help(),
		Tags:  m.Tags(),
		Value: v,
	}
}

type EventBackend struct {
	Events []Event
}

func (b *EventBackend) Close() error { return nil }

func (b *EventBackend) Set(m Metric, v float64) { b.call(m, v) }

func (b *EventBackend) Add(m Metric, v float64) { b.call(m, v) }

func (b *EventBackend) Observe(m Metric, v time.Duration) { b.call(m, v) }

func (b *EventBackend) call(m Metric, v interface{}) { b.Events = append(b.Events, MakeEvent(m, v)) }
