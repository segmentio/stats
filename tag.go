package stats

import (
	statsv5 "github.com/segmentio/stats/v5"
)

// Tag behaves like [stats/v5.Tag].
type Tag = statsv5.Tag

// T behaves like [stats/v5.T].
func T(k, v string) Tag {
	return statsv5.T(k, v)
}

// M behaves like [stats/v5.M].
func M(m map[string]string) []Tag {
	return statsv5.M(m)
}

// TagsAreSorted behaves like [stats/v5.TagsAreSorted].
func TagsAreSorted(tags []Tag) bool {
	return statsv5.TagsAreSorted(tags)
}

// SortTags behaves like [stats/v5.SortTags].
func SortTags(tags []Tag) []Tag {
	return statsv5.SortTags(tags)
}
