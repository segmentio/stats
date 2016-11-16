package datadog

import (
	"strconv"

	"github.com/segmentio/stats"
)

func appendMetric(b []byte, m stats.Metric) []byte {
	b = append(b, m.Name...)
	b = append(b, ':')
	b = strconv.AppendFloat(b, m.Value, 'g', -1, 64)
	b = append(b, '|')

	switch m.Type {
	case stats.CounterType:
		b = append(b, 'c')

	case stats.GaugeType:
		b = append(b, 'g')

	default:
		b = append(b, '?') // unsupported
	}

	if m.Sample > 1 {
		b = append(b, '|', '@')
		b = strconv.AppendFloat(b, 1/float64(m.Sample), 'g', -1, 64)
	}

	if len(m.Tags) != 0 {
		b = append(b, '|', '#')

		for i, t := range m.Tags {
			if i != 0 {
				b = append(b, ',')
			}
			b = append(b, t.Name...)
			b = append(b, ':')
			b = append(b, t.Value...)
		}
	}

	return append(b, '\n')
}
