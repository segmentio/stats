package stats

import "sort"

type Tag struct {
	Name  string
	Value string
}

type tags []Tag

func (t tags) Less(i int, j int) bool {
	t1 := t[i]
	t2 := t[j]

	if t1.Name < t2.Name {
		return true
	}

	if t1.Name > t2.Name {
		return false
	}

	return t1.Value < t2.Value
}

func (t tags) Swap(i int, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t tags) Len() int {
	return len(t)
}

func sortTags(t []Tag) []Tag {
	sort.Sort(tags(t))
	return t
}

func concatTags(t1 []Tag, t2 []Tag) []Tag {
	if len(t1) == 0 {
		return copyTags(t2)
	}

	if len(t2) == 0 {
		return copyTags(t1)
	}

	n1 := len(t1)
	n2 := len(t2)
	t3 := make([]Tag, n1+n2)
	copy(t3, t1)
	copy(t3[n1:], t2)
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
