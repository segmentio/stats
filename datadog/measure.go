package datadog

import (
	"math"
	"strconv"
	"strings"

	"github.com/segmentio/stats/v4"
)

// Datagram format: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell

// AppendMeasure is a formatting routine to append the dogstatsd protocol
// representation of a measure to a memory buffer.
// Tags listed in the s.filters are removed. (some tags may not be suitable for submission to DataDog)
// Histogram metrics will be sent as distribution type if the metric name matches s.distPrefixes
func (s *serializer) AppendMeasure(b []byte, m stats.Measure) []byte {
	for _, field := range m.Fields {
		b = append(b, m.Name...)
		if len(field.Name) != 0 {
			b = append(b, '.')
			b = append(b, field.Name...)
		}
		b = append(b, ':')

		switch v := field.Value; v.Type() {
		case stats.Bool:
			if v.Bool() {
				b = append(b, '1')
			} else {
				b = append(b, '0')
			}
		case stats.Int:
			b = strconv.AppendInt(b, v.Int(), 10)
		case stats.Uint:
			b = strconv.AppendUint(b, v.Uint(), 10)
		case stats.Float:
			b = strconv.AppendFloat(b, normalizeFloat(v.Float()), 'g', -1, 64)
		case stats.Duration:
			b = strconv.AppendFloat(b, v.Duration().Seconds(), 'g', -1, 64)
		default:
			b = append(b, '0')
		}

		switch field.Type() {
		case stats.Counter:
			b = append(b, '|', 'c')
		case stats.Gauge:
			b = append(b, '|', 'g')
		default:
			if s.sendDist(field.Name) {
				b = append(b, '|', 'd')
			} else {
				b = append(b, '|', 'h')
			}
		}

		if n := len(m.Tags); n != 0 {
			b = append(b, '|', '#')

			for i, t := range m.Tags {
				if _, ok := s.filters[t.Name]; !ok {
					if i != 0 {
						b = append(b, ',')
					}
					b = append(b, t.Name...)
					b = append(b, ':')
					b = append(b, t.Value...)
				}
			}
		}

		b = append(b, '\n')
	}

	return b
}

func normalizeFloat(f float64) float64 {
	switch {
	case math.IsNaN(f):
		return 0.0
	case math.IsInf(f, +1):
		return +math.MaxFloat64
	case math.IsInf(f, -1):
		return -math.MaxFloat64
	default:
		return f
	}
}

func (s *serializer) sendDist(name string) bool {
	if s.distPrefixes == nil {
		return false
	}
	for _, prefix := range s.distPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
