package prometheus

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/segmentio/stats"
)

type Labels []Label

func makeLabels(tags stats.Tags) (labels Labels) {
	if tags != nil {
		labels = make(Labels, len(tags))

		for i, tag := range tags {
			labels[i] = Label{
				Name:  tag.Name,
				Value: tag.Value,
			}
		}
	}
	return
}

func (labels Labels) Format(f fmt.State, r rune) {
	if len(labels) != 0 {
		io.WriteString(f, "{")

		for i, label := range labels {
			if i != 0 {
				io.WriteString(f, ", ")
			}
			label.Format(f, r)
		}

		io.WriteString(f, "}")
	}
}

func (labels Labels) String() string { return fmt.Sprint(labels) }

type Label struct {
	Name  string
	Value string
}

func (label Label) Format(f fmt.State, _ rune) {
	io.WriteString(f, sanitize(label.Name))
	io.WriteString(f, `="`)
	io.WriteString(f, escape(label.Value))
	io.WriteString(f, `"`)
}

func (label Label) String() string { return fmt.Sprint(label) }

func sanitize(s string) string {
	if isSafeString(s) {
		// fast path
		return s
	}

	b := &bytes.Buffer{}
	b.Grow(len(s))

	for i := range s {
		if c := s[i]; isSafeByte(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('_')
		}
	}

	return b.String()
}

func escape(s string) string {
	if strings.IndexByte(s, '\n') < 0 && strings.IndexByte(s, '"') < 0 && strings.IndexByte(s, '\\') < 0 {
		// fast path
		return s
	}

	b := &bytes.Buffer{}
	b.Grow(len(s) + 10)

	for i := range s {
		switch c := s[i]; c {
		case '\n':
			b.WriteString(`\n`)
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteByte(c)
		}
	}

	return b.String()
}

func isSafeString(s string) bool {
	for i := range s {
		if !isSafeByte(s[i]) {
			return false
		}
	}
	return true
}

func isSafeByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= 'Z') || b == '_' || b == ':'
}
