package prometheus

import (
	"github.com/segmentio/fasthash/jody"
	"github.com/vertoforce/stats"
)

type label struct {
	name  string
	value string
}

func (l label) hash() uint64 {
	h := jody.Init64
	h = jody.AddString64(h, l.name)
	h = jody.AddString64(h, l.value)
	return h
}

func (l label) equal(other label) bool {
	return l.name == other.name && l.value == other.value
}

func (l label) less(other label) bool {
	return l.name < other.name || (l.name == other.name && l.value < other.value)
}

type labels []label

func makeLabels(l ...label) labels {
	m := make(labels, len(l))
	copy(m, l)
	return m
}

func (l labels) copyAppend(m ...label) labels {
	c := make(labels, 0, len(l)+len(m))
	c = append(c, l...)
	c = append(c, m...)
	return c
}

func (l labels) copy() labels {
	return makeLabels(l...)
}

func (l labels) hash() uint64 {
	h := jody.Init64

	for i := range l {
		h = jody.AddString64(h, l[i].name)
		h = jody.AddString64(h, l[i].value)
	}

	return h
}

func (l labels) equal(other labels) bool {
	if len(l) != len(other) {
		return false
	}
	for i := range l {
		if !l[i].equal(other[i]) {
			return false
		}
	}
	return true
}

func (l labels) less(other labels) bool {
	n1 := len(l)
	n2 := len(other)

	for i := 0; i != n1 && i != n2; i++ {
		if !l[i].equal(other[i]) {
			return l[i].less(other[i])
		}
	}

	return n1 < n2
}

func (l labels) appendTags(tags ...stats.Tag) labels {
	for _, t := range tags {
		l = append(l, label{name: t.Name, value: t.Value})
	}
	return l
}
