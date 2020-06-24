package topk

import (
	"fmt"
	"github.com/segmentio/stats/v4"
	boom "github.com/tylertreat/BoomFilters"
	"strings"
	"time"
)

type Config struct {
	Engine *stats.Engine

	// TODO: add more configuration
	// k: number of elements to track
	// flushInterval: frequency to flush topk metrics
}

func NewHandlerWith(config Config) *Engine {
	topk := boom.NewTopK(0.001, 0.99, 5)

	e := &Engine{
		eng:  config.Engine,
		topk: topk,
	}
	go e.scheduleFlush()
	return e
}

type Engine struct {
	eng  *stats.Engine
	topk *boom.TopK // TODO: validate that this is safe for concurrent use
}

// Incr increments by one the counter identified by name and tags.
func (e *Engine) Incr(name string, tags ...stats.Tag) {
	if len(tags) != 0 && !stats.TagsAreSorted(tags) {
		stats.SortTags(tags)
	}

	stringTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		stringTags = append(stringTags, tag.String())
	}

	concatTags := strings.Join(stringTags, ",")

	// keys are laid out as "name:tag1,tag2,tag3"
	key := fmt.Sprintf("%s:%s", name, concatTags)

	e.topk.Add([]byte(key))
}

// emit runs on a timer
func (e *Engine) scheduleFlush() {
	ticker := time.NewTicker(1 * time.Minute)

	// TODO: add a shutdown mechanism?
	go func() {
		for {
			select {
			case <-ticker.C:
				e.flush()
			}
		}
	}()
}

func (e *Engine) flush() {
	for _, element := range e.topk.Elements() {
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
	e.topk.Reset()
}

// TODO: decorate other methods
