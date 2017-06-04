package prometheus

import (
	"strconv"
	"strings"
)

func appendMetric(b []byte, metric metric) []byte {
	if len(metric.help) != 0 {
		b = append(b, "# HELP "...)
		b = append(b, metric.name...)
		b = append(b, ' ')
		b = appendEscapedString(b, metric.help, indexOfSpecialHelpByte)
		b = append(b, '\n')
	}

	if metric.mtype != untyped {
		b = append(b, "# TYPE "...)
		b = append(b, metric.name...)
		b = append(b, ' ')
		b = append(b, metric.mtype.String()...)
		b = append(b, '\n')
	}

	b = append(b, metric.name...)
	b = appendLabels(b, metric.labels...)
	b = append(b, ' ')
	b = strconv.AppendFloat(b, metric.value, 'g', -1, 64)

	if !metric.time.IsZero() {
		t := metric.time.Unix() * 1000
		t += int64(metric.time.Nanosecond() / 1e6) // millisecond
		b = append(b, ' ')
		b = strconv.AppendInt(b, t, 10)
	}

	return append(b, '\n')
}

func appendLabels(b []byte, labels ...label) []byte {
	if len(labels) != 0 {
		b = append(b, '{')

		for i, label := range labels {
			if i != 0 {
				b = append(b, ',')
			}
			b = appendLabel(b, label)
		}

		b = append(b, '}')
	}
	return b
}

func appendLabel(b []byte, label label) []byte {
	b = append(b, label.name...)
	b = append(b, '=', '"')
	b = appendEscapedString(b, label.value, indexOfSpecialLabelValueByte)
	return append(b, '"')
}

func appendEscapedString(b []byte, s string, indexOfSpecialByte func(string) int) []byte {
	i := 0
	n := len(s)

	for i != n {
		j := i + indexOfSpecialByte(s[i:])
		b = append(b, s[i:j]...)

		if i = j; i != n {
			switch c := s[i]; c {
			case '\n':
				b = append(b, '\\', 'n')
			default:
				b = append(b, '\\', c)
			}
			i++
		}
	}

	return b
}

func indexOfSpecialHelpByte(s string) int {
	return indexOf(s, '\\', '\n')
}

func indexOfSpecialLabelValueByte(s string) int {
	return indexOf(s, '\\', '"', '\n')
}

func indexOf(s string, bytes ...byte) int {
	i := len(s)

	for _, b := range bytes {
		if j := strings.IndexByte(s, b); j >= 0 && j < i {
			i = j
		}
	}

	return i
}
