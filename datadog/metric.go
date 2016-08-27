package datadog

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/iostats"
)

type MetricType string

const (
	Gauge     MetricType = "g"
	Counter   MetricType = "c"
	Histogram MetricType = "h"
	Set       MetricType = "s"
)

type Metric struct {
	Name       string
	Value      float64
	Type       MetricType
	SampleRate float64
	Tags       stats.Tags
}

func ParseMetric(s string) (m Metric, err error) {
	var next = strings.TrimSpace(s)
	var name string
	var val string
	var typ string
	var rate string
	var tags string

	val, next = nextToken(next)
	typ, next = nextToken(next)
	rate, tags = nextToken(next)
	name, val = split(val, ':')

	if len(name) == 0 {
		err = fmt.Errorf("datadog: %#v is missing a metric name", s)
		return
	}

	if len(val) == 0 {
		err = fmt.Errorf("datadog: %#v is missing a metric value", s)
		return
	}

	if len(typ) == 0 {
		err = fmt.Errorf("datadog: %#v is missing a metric type", s)
		return
	}

	if len(rate) != 0 {
		switch rate[0] {
		case '#': // no sample rate, just tags
			rate, tags = "", rate
		case '@':
			rate = rate[1:]
		default:
			err = fmt.Errorf("datadog: %#v has a malformed sample rate", s)
			return
		}
	}

	if len(tags) != 0 {
		switch tags[0] {
		case '#':
			tags = tags[1:]
		default:
			err = fmt.Errorf("datadog: %#v has malformed tags", s)
			return
		}
	}

	var value float64
	var sampleRate float64

	if value, err = strconv.ParseFloat(val, 64); err != nil {
		err = fmt.Errorf("datadog: %#v has a malformed value", s)
		return
	}

	if len(rate) != 0 {
		if sampleRate, err = strconv.ParseFloat(rate, 64); err != nil {
			err = fmt.Errorf("datadog: %#v has a malformed sample rate", s)
			return
		}
	}

	if sampleRate == 0 {
		sampleRate = 1
	}

	m = Metric{
		Name:       name,
		Value:      value,
		Type:       MetricType(typ),
		SampleRate: sampleRate,
	}

	if tags := strings.Split(tags, ","); len(tags) != 0 {
		m.Tags = make(stats.Tags, 0, len(tags))

		for _, tag := range tags {
			if tag = strings.TrimSpace(tag); len(tag) != 0 {
				name, value := split(tag, ':')
				m.Tags = append(m.Tags, stats.Tag{Name: name, Value: value})
			}
		}
	}

	return
}

func nextToken(s string) (token string, next string) {
	if off := strings.IndexByte(s, '|'); off >= 0 {
		token, next = s[:off], s[off+1:]
	} else {
		token = s
	}
	return
}

func split(s string, b byte) (head string, tail string) {
	if off := strings.LastIndexByte(s, b); off >= 0 {
		head, tail = s[:off], s[off+1:]
	} else {
		head = s
	}
	return
}

func (m Metric) Write(w io.Writer) (err error) {
	f := &iostats.Formatter{}
	defer f.Release()
	return m.write(w, f)
}

func (m Metric) write(w io.Writer, f *iostats.Formatter) (err error) {
	defer func() { err = convertPanicToError(recover()) }()

	write(w, f.FormatStringFunc(m.Name, sanitizeRune))
	write(w, f.FormatByte(':'))
	write(w, f.FormatFloat(m.Value, 'g', -1, 64))
	write(w, f.FormatByte('|'))
	write(w, f.FormatString(string(m.Type)))

	if r := float64(m.SampleRate); r != 1 {
		write(w, f.FormatString("|@"))
		write(w, f.FormatFloat(r, 'g', -1, 64))
	}

	if len(m.Tags) != 0 {
		write(w, f.FormatString("|#"))

		for i, t := range m.Tags {
			if i != 0 {
				write(w, f.FormatByte(','))
			}
			write(w, f.FormatStringFunc(t.Name, sanitizeRune))
			write(w, f.FormatByte(':'))
			write(w, f.FormatStringFunc(t.Value, sanitizeRune))
		}
	}

	write(w, f.FormatByte('\n'))
	return
}

func (m Metric) String() string {
	b := &bytes.Buffer{}
	b.Grow(64)
	m.Write(b)
	return b.String()
}

func sanitizeRune(r rune) rune {
	switch r {
	case ',', ':', '|', '@', '#':
		return '_'
	}
	return r
}

func write(w io.Writer, b []byte) {
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
}

func convertPanicToError(v interface{}) error {
	if v == nil {
		return nil
	}
	return v.(error)
}
