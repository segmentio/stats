package otlp

import (
	"hash/maphash"
	"sort"
	"time"

	"github.com/vertoforce/stats"
)

type metric struct {
	measureName string
	fieldName   string
	fieldType   stats.FieldType
	flushed     bool
	time        time.Time
	value       stats.Value
	sum         float64
	sign        uint64
	count       uint64
	buckets     metricBuckets
	tags        []stats.Tag
}

func (m *metric) signature() uint64 {
	h := maphash.Hash{}
	h.SetSeed(hashseed)
	h.WriteString(m.measureName)
	h.WriteString(m.fieldName)

	sort.Slice(m.tags, func(i, j int) bool {
		return m.tags[i].Name > m.tags[j].Name
	})

	for _, tag := range m.tags {
		h.WriteString(tag.String())
	}

	return h.Sum64()
}

func (m *metric) add(v stats.Value) stats.Value {
	switch v.Type() {
	case stats.Int:
		return stats.ValueOf(m.value.Int() + v.Int())
	case stats.Uint:
		return stats.ValueOf(m.value.Uint() + v.Uint())
	case stats.Float:
		return stats.ValueOf(m.value.Float() + v.Float())
	}
	return v
}

type bucket struct {
	count      uint64
	upperBound float64
}

type metricBuckets []bucket

func makeMetricBuckets(buckets []stats.Value) metricBuckets {
	b := make(metricBuckets, len(buckets))
	for i := range buckets {
		b[i].upperBound = valueOf(buckets[i])
	}
	return b
}

func (b metricBuckets) update(v float64) {
	for i := range b {
		if v <= b[i].upperBound {
			b[i].count++
			break
		}
	}
}
