package influxdb

import (
	"strconv"
	"time"

	"github.com/segmentio/stats"
)

// AppendMeasure is a formatting routine to append the InflxDB line protocol
// representation of a measure to a memory buffer.
func AppendMeasure(b []byte, t time.Time, m stats.Measure) []byte {
	b = append(b, m.Name...)

	for _, tag := range m.Tags {
		b = append(b, ',')
		b = append(b, tag.Name...)
		b = append(b, '=')
		b = append(b, tag.Value...)
	}

	for i, field := range m.Fields {
		if len(field.Name) == 0 {
			field.Name = "value"
		}

		if i == 0 {
			b = append(b, ' ')
		} else {
			b = append(b, ',')
		}

		b = append(b, field.Name...)
		b = append(b, '=')

		switch v := field.Value; v.Type() {
		case stats.Null:
		case stats.Bool:
			if v.Bool() {
				b = append(b, "true"...)
			} else {
				b = append(b, "false"...)
			}
		case stats.Int:
			b = strconv.AppendInt(b, v.Int(), 10)
		case stats.Uint:
			b = strconv.AppendUint(b, v.Uint(), 10)
		case stats.Float:
			b = strconv.AppendFloat(b, v.Float(), 'g', -1, 64)
		case stats.Duration:
			b = strconv.AppendFloat(b, v.Duration().Seconds(), 'g', -1, 64)
		}
	}

	b = append(b, ' ')
	b = strconv.AppendInt(b, t.UnixNano(), 10)

	return append(b, '\n')
}
