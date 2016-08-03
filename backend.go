package stats

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
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
	return &backend{out: out, enc: enc}
}

type backend struct {
	mtx sync.Mutex
	out *bufio.Writer
	enc *json.Encoder
}

func (b *backend) Close() error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.out.Flush()
}

func (b *backend) Set(m Metric, v float64) error {
	return b.send("gauge", m, v)
}

func (b *backend) Add(m Metric, v float64) error {
	return b.send("counter", m, v)
}

func (b *backend) Observe(m Metric, v time.Duration) error {
	return b.send("histogram", m, v.Seconds())
}

func (b *backend) send(t string, m Metric, v float64) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.enc.Encode(struct {
		Type  string  `json:"type"`
		Name  string  `json:"name"`
		Help  string  `json:"help,omitempty"`
		Value float64 `json:"value"`
		Tags  Tags    `json:"tags,omitempty"`
	}{
		Type:  t,
		Name:  m.Name(),
		Help:  m.Help(),
		Value: v,
		Tags:  m.Tags(),
	})
}

func MultiBackend(backends ...Backend) Backend {
	return multiBackend(backends)
}

type multiBackend []Backend

func (b multiBackend) Close() (err error) {
	for _, x := range b {
		err = appendError(err, x.Close())
	}
	return
}

func (b multiBackend) Set(m Metric, v float64) (err error) {
	for _, x := range b {
		err = appendError(err, x.Set(m, v))
	}
	return
}

func (b multiBackend) Add(m Metric, v float64) (err error) {
	for _, x := range b {
		err = appendError(err, x.Add(m, v))
	}
	return
}

func (b multiBackend) Observe(m Metric, v time.Duration) (err error) {
	for _, x := range b {
		err = appendError(err, x.Observe(m, v))
	}
	return
}
