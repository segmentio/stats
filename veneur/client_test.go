package veneur

import (
	"testing"
	"time"

	"github.com/segmentio/stats"
)

func TestClient(t *testing.T) {
	client := NewClientWith(ClientConfig{
		GlobalOnly: true,
		SinksOnly:  []string{SignalfxSink},
	})

	for i := 0; i != 1000; i++ {
		client.HandleMeasures(time.Time{}, stats.Measure{
			Name: "request",
			Fields: []stats.Field{
				{Name: "count", Value: stats.ValueOf(5)},
				{Name: "rtt", Value: stats.ValueOf(100 * time.Millisecond)},
			},
			Tags: []stats.Tag{
				TagSignalfxOnly,
			},
		})
	}

	if err := client.Close(); err != nil {
		t.Error(err)
	}
}
