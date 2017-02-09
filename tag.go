package stats

import "sort"

// T returns a tag with `name`, `value`.
// T is just sugar for `Tag{}`.
func T(name, value string) Tag {
	return Tag{
		Name:  name,
		Value: value,
	}
}

// Tag represents a single tag that can be set on a metric.
type Tag struct {
	Name  string
	Value string
}

// RawTags are a list of tags in a serialized from that can be used to construct
// a metric key.
//
// This is a low-level API that is intended to be used for optimization purposes
// and most application should not need it.
type RawTags string

// MakeRawTags converts a slice of tags to their RawTags representation.
func MakeRawTags(tags []Tag) RawTags {
	return RawTags(appendTags(make([]byte, 0, tagsLen(tags)), tags))
}

type tags []Tag

func (t tags) Less(i int, j int) bool {
	t1 := t[i]
	t2 := t[j]
	return t1.Name < t2.Name || (t1.Name == t2.Name && t1.Value < t2.Value)
}

func (t tags) Swap(i int, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tags) Len() int {
	return len(t)
}

func sortTags(t []Tag) {
	sort.Sort(tags(t))
}

func concatTags(t1 []Tag, t2 []Tag) []Tag {
	t3 := make([]Tag, 0, len(t1)+len(t2))
	t3 = append(t3, t1...)
	t3 = append(t3, t2...)
	return t3
}

func copyTags(tags []Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}
	ctags := make([]Tag, len(tags))
	copy(ctags, tags)
	return ctags
}

func appendTags(b []byte, tags []Tag) []byte {
	for i, t := range tags {
		if i != 0 {
			b = append(b, '&')
		}
		b = append(b, t.Name...)
		b = append(b, '=')
		b = append(b, t.Value...)
	}
	return b
}

func tagsLen(tags []Tag) (n int) {
	if len(tags) != 0 {
		for _, t := range tags {
			n += len(t.Name) + len(t.Value) + 2
		}
		n--
	}
	return
}
