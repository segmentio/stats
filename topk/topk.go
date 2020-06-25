package topk

import (
	"sync"

	boom "github.com/tylertreat/BoomFilters"
)

type topk struct {
	_topk *boom.TopK
	mut   sync.Mutex
}

func (t *topk) add(b []byte) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t._topk.Add(b)
}

func (t *topk) elements() []*boom.Element {
	t.mut.Lock()
	defer t.mut.Unlock()
	return t._topk.Elements()
}

func (t *topk) reset() {
	t.mut.Lock()
	defer t.mut.Unlock()
	t._topk.Reset()
}
