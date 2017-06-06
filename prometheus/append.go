package prometheus

import (
	"strconv"
	"strings"
)

func appendMetricName(b []byte, s string) []byte {
	i := len(b)
	b = append(b, s...)
	replaceInvalidMetricBytes(b[i:])
	return b
}

func appendMetricScopedName(b []byte, scope string, name string) []byte {
	if len(scope) != 0 {
		b = appendMetricName(b, scope)
		b = append(b, '_')
	}
	return appendMetricName(b, name)
}

func appendMetric(b []byte, metric metric) []byte {
	if len(metric.help) != 0 {
		b = appendMetricHelp(b, metric.scope, metric.rootName(), metric.help)
	}

	if metric.mtype != untyped {
		b = appendMetricType(b, metric.scope, metric.rootName(), metric.mtype.String())
	}

	b = appendMetricScopedName(b, metric.scope, metric.name)
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

func appendMetricHelp(b []byte, scope string, name string, help string) []byte {
	b = append(b, "# HELP "...)
	b = appendMetricScopedName(b, scope, name)
	b = append(b, ' ')
	b = appendEscapedString(b, help, indexOfSpecialHelpByte)
	return append(b, '\n')
}

func appendMetricType(b []byte, scope string, name string, mtype string) []byte {
	b = append(b, "# TYPE "...)
	b = appendMetricScopedName(b, scope, name)
	b = append(b, ' ')
	b = append(b, mtype...)
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
	b = appendLabelName(b, label.name)
	b = append(b, '=', '"')
	b = appendEscapedString(b, label.value, indexOfSpecialLabelValueByte)
	return append(b, '"')
}

func appendLabelName(b []byte, s string) []byte {
	i := len(b)
	b = append(b, s...)
	replaceInvalidLabelBytes(b[i:])
	return b
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

func replaceInvalidMetricBytes(b []byte) {
	if len(b) != 0 {
		if !isValidFirstMetricByte(b[0]) {
			b[0] = '_'
		}
		c := b[1:]
		for i := range c {
			if !isValidMetricByte(c[i]) {
				c[i] = '_'
			}
		}
	}
}

func replaceInvalidLabelBytes(b []byte) {
	if len(b) != 0 {
		if !isValidFirstLabelByte(b[0]) {
			b[0] = '_'
		}
		c := b[1:]
		for i := range c {
			if !isValidLabelByte(c[i]) {
				c[i] = '_'
			}
		}
	}
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isAlpha(c byte) bool {
	return isLower(c) || isUpper(c)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlphanum(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

func isValidFirstMetricByte(c byte) bool {
	return isAlpha(c) || (c == '_') || (c == ':')
}

func isValidMetricByte(c byte) bool {
	return isAlphanum(c) || (c == '_') || (c == ':')
}

func isValidFirstLabelByte(c byte) bool {
	return isAlpha(c) || (c == '_')
}

func isValidLabelByte(c byte) bool {
	return isAlphanum(c) || (c == '_')
}
