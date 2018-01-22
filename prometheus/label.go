package prometheus

import (
	"github.com/segmentio/fasthash/jody"
	"github.com/segmentio/stats"
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

func (l1 label) equal(l2 label) bool {
	return l1.name == l2.name && l1.value == l2.value
}

func (l1 label) less(l2 label) bool {
	return l1.name < l2.name || (l1.name == l2.name && l1.value < l2.value)
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

func (l1 labels) equal(l2 labels) bool {
	if len(l1) != len(l2) {
		return false
	}
	for i := range l1 {
		if !l1[i].equal(l2[i]) {
			return false
		}
	}
	return true
}

func (l1 labels) less(l2 labels) bool {
	n1 := len(l1)
	n2 := len(l2)

	for i := 0; i != n1 && i != n2; i++ {
		if !l1[i].equal(l2[i]) {
			return l1[i].less(l2[i])
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
