package stats

import (
	"fmt"
	"sync"
	"time"
)

type Event struct {
	Type  string    `json:"type"`
	Name  string    `json:"name"`
	Value float64   `json:"value"`
	Time  time.Time `json:"time"`
	Tags  Tags      `json:"tags,omitempty"`
}

func MakeEvent(m Metric, v float64, t time.Time) Event {
	return Event{
		Type:  m.Type(),
		Name:  m.Name(),
		Tags:  m.Tags(),
		Value: v,
		Time:  t,
	}
}

func (e Event) String() string {
	return fmt.Sprint(e)
}

func (e Event) Format(f fmt.State, _ rune) {
	fmt.Fprintf(f, "{ type: %#v, name: %#v, value: %g, time: \"%s\", tags: [%v] }",
		e.Type,
		e.Name,
		e.Value,
		e.Time,
		e.Tags,
	)
}

type EventBackend struct {
	sync.RWMutex
	Events []Event
}

func (b *EventBackend) Close() error { return nil }

func (b *EventBackend) Set(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *EventBackend) Add(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *EventBackend) Observe(m Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *EventBackend) call(m Metric, v float64, t time.Time) {
	b.Lock()
	defer b.Unlock()
	b.Events = append(b.Events, MakeEvent(m, v, t))
}
