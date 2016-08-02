package stats

import (
	"bufio"
	"encoding/json"
	"io"
)

type Client interface {
	io.Closer

	NewTracker(namespace string) Tracker
}

func NewClient(w io.Writer) Client {
	return client{w: bufio.NewWriter(w)}
}

type client struct {
	w *bufio.Writer
}

func (c client) Close() error {
	return c.w.Flush()
}

func (c client) NewTracker(namespace string) Tracker {
	return TrackerFunc(func(m Metric, v Value) {
		json.NewEncoder(c.w).Encode(struct {
			Name  string  `json:"name"`
			Type  string  `json:"type"`
			Value float64 `json:"value"`
			Tags  Tags    `json:"tags,omitempty"`
		}{
			Name:  namespace + "." + m.Name() + "." + v.Measure(),
			Type:  v.Type(),
			Value: v.Value(),
			Tags:  m.Tags(),
		})
	})
}
