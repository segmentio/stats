package stats

// Tag represents a single tag that can be set on a metric.
type Tag struct {
	Name  string
	Value string
}

// T returns a new tag made of name and value.
func T(name string, value string) Tag {
	return Tag{
		Name:  name,
		Value: value,
	}
}

func concatTags(t1 []Tag, t2 []Tag) []Tag {
	n := len(t1) + len(t2)
	if n == 0 {
		return nil
	}
	t3 := make([]Tag, 0, n)
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
