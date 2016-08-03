package stats

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

type Backend interface {
	io.Closer

	Set(Metric, float64) error

	Add(Metric, float64) error

	Observe(Metric, time.Duration) error
}

func NewBackend(w io.Writer) Backend {
	out := bufio.NewWriter(w)
	enc := json.NewEncoder(out)
	return backend{out: out, enc: enc}
}

type backend struct {
	out *bufio.Writer
	enc *json.Encoder
}

func (b backend) Close() error {
	return b.out.Flush()
}

func (b backend) Set(m Metric, x float64) error {
	return b.send("gauge", m, x)
}

func (b backend) Add(m Metric, x float64) error {
	return b.send("counter", m, x)
}

func (b backend) Observe(m Metric, x time.Duration) error {
	return b.send("histogram", m, x.Seconds())
}

func (b backend) send(t string, m Metric, v float64) error {
	return b.enc.Encode(struct {
		Type  string      `json:"type"`
		Name  string      `json:"name"`
		Help  string      `json:"help,omitempty"`
		Value interface{} `json:"value"`
		Tags  Tags        `json:"tags,omitempty"`
	}{
		Type:  t,
		Name:  m.Name(),
		Help:  m.Help(),
		Value: v,
		Tags:  m.Tags(),
	})
}
