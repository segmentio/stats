package jsonstats

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/segmentio/stats"
)

func NewBackend(w io.Writer) stats.Backend {
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
	return b.out.Flush()
}

func (b *backend) Set(m stats.Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *backend) Add(m stats.Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *backend) Observe(m stats.Metric, v float64, t time.Time) { b.call(m, v, t) }

func (b *backend) call(m stats.Metric, v float64, t time.Time) {
	e := stats.MakeEvent(m, v, t)
	b.mtx.Lock()
	b.enc.Encode(e)
	b.mtx.Unlock()
}
