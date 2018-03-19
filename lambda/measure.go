package lambda

import (
	"strconv"
	"time"

	"github.com/segmentio/stats"
)

// Lambda metrics format : MONITORING|<unix_epoch_timestamp>|<value>|<metric_type>|<metric_name>|#<tag_list>
func AppendMeasure(b []byte, t time.Time, m stats.Measure) []byte {
	tags := formatTags(m.Tags)

	for _, field := range m.Fields {
		b = append(b, "MONITORING|"...)
		b = strconv.AppendInt(b, t.UnixNano(), 10)
		b = append(b, '|')
		switch v := field.Value; v.Type() {
		case stats.Int:
			b = strconv.AppendInt(b, v.Int(), 10)
		case stats.Uint:
			b = strconv.AppendUint(b, v.Uint(), 10)
		case stats.Float:
			b = strconv.AppendFloat(b, v.Float(), 'g', -1, 64)
		}
		b = append(b, '|')
		switch field.Type() {
		case stats.Counter:
			b = append(b, "count"...)
		case stats.Gauge:
			b = append(b, "gauge"...)
		case stats.Histogram:
			b = append(b, "histogram"...)
		}
		b = append(b, '|')
		if len(field.Name) == 0 {
			b = append(b, m.Name...)
		} else {
			b = append(b, (m.Name + "." + field.Name)...)
		}
		if len(m.Tags) > 0 {
			b = append(b, '|')
			b = append(b, tags...)
		}
		b = append(b, '\n')
	}
	return b
}

func formatTags(tags []stats.Tag) []byte {
	var b []byte

	b = append(b, '#')
	for _, tag := range tags {
		if len(b) > 1 {
			b = append(b, ',')
		}
		b = append(b, tag.Name...)
		b = append(b, '=')
		b = append(b, tag.Value...)
	}

	return b
}
