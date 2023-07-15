package stats

import (
	"sync"

	"golang.org/x/exp/slices"
)

// A Tag is a pair of a string key and value set on measures to define the
// dimensions of the metrics.
type Tag struct {
	Name  string
	Value string
}

// T is shorthand for `stats.Tag{Name: "blah", Value: "foo"}`  It returns
// the tag for Name k and Value v
func T(k, v string) Tag {
	return Tag{Name: k, Value: v}
}

func (t Tag) String() string {
	return t.Name + "=" + t.Value
}

// M allows for creating a tag list from a map.
func M(m map[string]string) []Tag {
	tags := make([]Tag, 0, len(m))
	for k, v := range m {
		tags = append(tags, T(k, v))
	}
	return tags
}

// TagsAreSorted returns true if the given list of tags is sorted by tag name,
// false otherwise.
func TagsAreSorted(tags []Tag) bool {
	return slices.IsSortedFunc(tags, tagIsLess)
}

// SortTags sorts and deduplicates tags in-place,
// favoring later elements whenever a tag name duplicate occurs.
// The returned slice may be shorter than the input due to duplicates.
func SortTags(tags []Tag) []Tag {
	// Stable sort ensures that we have deterministic
	// "latest wins" deduplication.
	// For 20 or fewer tags, this is as fast as an unstable sort.
	slices.SortStableFunc(tags, tagIsLess)

	return deduplicateTags(tags)
}

func tagIsLess(a, b Tag) bool { return a.Name < b.Name }

func deduplicateTags(tags []Tag) []Tag {
	var prev string
	out := tags[:0]

	for _, tag := range tags {
		switch {
		case tag.Name == "":
			// Ignore unnamed tags.
			continue

		case tag.Name != prev:
			// Non-duplicate tag: keep.
			prev = tag.Name
			out = append(out, tag)

		default:
			// Duplicate tag: replace previous, same-named tag.
			i := len(out) - 1
			out[i] = tag
		}
	}

	if len(out) == 0 {
		// No input tags had non-empty names:
		// return nil to be consistent for ease of testing.
		return nil
	}

	return out
}

// mergeTags returns the sorted, deduplicated-by-name union of t1 and t2.
func mergeTags(t1, t2 []Tag) []Tag {
	n := len(t1) + len(t2)
	if n == 0 {
		return nil
	}

	out := make([]Tag, 0, n)
	out = append(out, t1...)
	out = append(out, t2...)

	return SortTags(out)
}

func copyTags(tags []Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}
	ctags := make([]Tag, len(tags))
	copy(ctags, tags)
	return ctags
}

type tagsBuffer struct {
	tags []Tag
}

func (b *tagsBuffer) reset() {
	for i := range b.tags {
		b.tags[i] = Tag{}
	}
	b.tags = b.tags[:0]
}

func (b *tagsBuffer) sort() {
	SortTags(b.tags)
}

func (b *tagsBuffer) append(tags ...Tag) {
	b.tags = append(b.tags, tags...)
}

var tagsPool = sync.Pool{
	New: func() any { return &tagsBuffer{tags: make([]Tag, 0, 8)} },
}
