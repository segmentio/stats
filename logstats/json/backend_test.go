package jsonstats

import (
	"bytes"
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestBackend(t *testing.T) {
	now := time.Unix(1, 0).In(time.UTC)

	b := &bytes.Buffer{}
	c := stats.NewClient("log", NewBackend(b), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").SetAt(1, now)
	c.Counter("events.count").AddAt(1, now)
	c.Histogram("events.seconds").ObserveAt(1, now)
	c.Close()

	if s := b.String(); s != `{"type":"gauge","name":"log.events.level","value":1,"tags":{"hello":"world"},"time":"1970-01-01T00:00:01Z"}
{"type":"counter","name":"log.events.count","value":1,"tags":{"hello":"world"},"time":"1970-01-01T00:00:01Z"}
{"type":"histogram","name":"log.events.seconds","value":1,"tags":{"hello":"world"},"time":"1970-01-01T00:00:01Z"}
` {
		t.Errorf("invalid json: %s", s)
	}
}
