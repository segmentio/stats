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
	c := stats.NewClient(NewBackend(b), stats.Tag{
		Name:  "hello",
		Value: "world",
	})

	c.Gauge("events.level").SetAt(1, now)
	c.Counter("events.count").AddAt(1, now)
	c.Histogram("events.seconds").ObserveAt(1, now)
	c.Close()

	if s := b.String(); s != `{"type":"gauge","name":"json.test.events.level","value":1,"sample":1,"time":"1970-01-01T00:00:01Z","tags":{"hello":"world"}}
{"type":"counter","name":"json.test.events.count","value":1,"sample":1,"time":"1970-01-01T00:00:01Z","tags":{"hello":"world"}}
{"type":"histogram","name":"json.test.events.seconds","value":1,"sample":1,"time":"1970-01-01T00:00:01Z","tags":{"hello":"world"}}
` {
		t.Errorf("invalid json: %s", s)
	}
}
