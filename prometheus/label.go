package prometheus

import "github.com/segmentio/stats"

type label struct {
	name  string
	value string
}

func (l label) hash() uint64 {
	h := offset64
	h = hashS(h, l.name)
	h = hashS(h, l.value)
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

func (l labels) appendTags(t ...stats.Tag) labels {
	c := cap(l)
	n := len(l)

	// This is an optimization to prevent making more than one dynamic memory
	// allocation.
	if (c - n) < len(t) {
		x := make(labels, n, n+len(t))
		copy(x, l)
		l = x
	}

	for i := range t {
		l = append(l, label{
			name:  t[i].Name,
			value: t[i].Value,
		})
	}
	return l
}

func (l labels) copy() labels {
	return makeLabels(l...)
}

func (l labels) hash() uint64 {
	h := offset64

	for i := range l {
		h = hashS(h, l[i].name)
		h = hashS(h, l[i].value)
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

func (l labels) Len() int {
	return len(l)
}

func (l labels) Swap(i int, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l labels) Less(i int, j int) bool {
	return l[i].less(l[j])
}
