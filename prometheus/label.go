package prometheus

import (
	"strconv"

	"github.com/segmentio/fasthash/jody"

	"github.com/segmentio/stats/v5"
)

type label struct {
	name  string
	value string
}

func (l label) equal(other label) bool {
	return l.name == other.name && l.value == other.value
}

func (l label) less(other label) bool {
	if l.name != other.name {
		return l.name < other.name
	}

	// Histogram bucket boundaries have to come out in increasing numeric
	// order. Comparing them as strings puts "+Inf" first ('+' is ASCII 43,
	// digits start at 48) and orders "10" ahead of "2".
	//
	// Scoped to "le" so that every other label keeps comparing as a string.
	if l.name == "le" {
		v1, err1 := strconv.ParseFloat(l.value, 64)
		v2, err2 := strconv.ParseFloat(other.value, 64)
		if err1 == nil && err2 == nil {
			return v1 < v2
		}
	}

	return l.value < other.value
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
