package topk

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/segmentio/stats/v4"
	boom "github.com/tylertreat/BoomFilters"
)

// delegate abstracts the *stats.Engine dependency
type delegate interface {
	Add(name string, value interface{}, tags ...stats.Tag)
}

type Engine struct {
	eng           delegate
	topk          topk
	flushInterval time.Duration
}

func NewHandler(ctx context.Context, opts ...Opt) (*Engine, error) {
	config := newTopkConfig()
	for _, opt := range opts {
		err := opt(&config)
		if err != nil {
			return nil, err
		}
	}
	boomTopk := boom.NewTopK(config.epsilon, config.delta, config.k)
	engine := &Engine{
		eng:           config.engine,
		topk:          topk{_topk: boomTopk},
		flushInterval: config.flushInterval,
	}
	go engine.flushLoop(ctx)
	return engine, nil
}

// Incr increments by one the counter identified by name and tags.
func (e *Engine) Incr(name string, tags ...stats.Tag) {
	if len(tags) != 0 && !stats.TagsAreSorted(tags) {
		stats.SortTags(tags)
	}
	tagString := tagsToString(tags...)
	// keys are laid out as "name:tag1,tag2,tag3"
	key := name + ":" + tagString
	e.topk.add([]byte(key))
}

// emit runs on a timer
func (e *Engine) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(e.flushInterval)
	defer e.flush()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.flush()
		}
	}
}

func (e *Engine) flush() {
	for _, element := range e.topk.elements() {
		parts := strings.Split(string(element.Data), ":")
		name := parts[0]

		tags := make([]stats.Tag, 0)
		for _, t := range strings.Split(parts[1], ",") {
			tagParts := strings.Split(t, "=")

			tags = append(tags, stats.Tag{
				Name:  tagParts[0],
				Value: tagParts[1],
			})
		}

		e.eng.Add(name, element.Freq, tags...)
	}

	// Restore to initial state.
	e.topk.reset()
}

func tagsToString(tags ...stats.Tag) string {
	// try to size the buffer within reason
	const tagStringExpectedSize = 20
	buf := bytes.NewBuffer(make([]byte, 0, len(tags)*tagStringExpectedSize))
	for i, tag := range tags {
		buf.WriteString(tag.String())
		if i < len(tags)-1 {
			buf.WriteString(",")
		}
	}
	return buf.String()
}

// TODO: decorate other methods
