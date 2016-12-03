package datadog

import (
	"strconv"

	"github.com/segmentio/stats"
)

func appendMetric(b []byte, m Metric) []byte {
	if len(m.Namespace.Name) != 0 {
		b = append(b, m.Namespace.Name...)
		b = append(b, '.')
	}

	b = append(b, m.Name...)
	b = append(b, ':')
	b = strconv.AppendFloat(b, m.Value, 'g', -1, 64)
	b = append(b, '|')
	b = append(b, m.Type...)

	if m.Rate != 0 && m.Rate != 1 {
		b = append(b, '|', '@')
		b = strconv.AppendFloat(b, m.Rate, 'g', -1, 64)
	}

	n1 := len(m.Namespace.Tags)
	n2 := len(m.Tags)

	if n1 != 0 || n2 != 0 {
		b = append(b, '|', '#')
		b = appendTags(b, m.Namespace.Tags)

		if n1 != 0 && n2 != 0 {
			b = append(b, ',')
		}

		b = appendTags(b, m.Tags)
	}

	return append(b, '\n')
}

func appendTags(b []byte, tags []stats.Tag) []byte {
	for i, t := range tags {
		if t.Name == "http_req_path" {
			// Datadog has complained numerous times that the request paths
			// generate too many custom metrics on their side, for now we'll
			// simply strip it out until we can come up with a better strategy
			// for handling those.
			continue
		}

		if i != 0 {
			b = append(b, ',')
		}

		b = append(b, t.Name...)
		b = append(b, ':')
		b = append(b, t.Value...)
	}
	return b
}
