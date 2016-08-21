package datadog

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/segmentio/stats"
)

type MetricType string

const (
	Gauge     MetricType = "g"
	Counter   MetricType = "c"
	Histogram MetricType = "h"
	Set       MetricType = "s"
)

type Metric struct {
	Name   string
	Value  float64
	Type   MetricType
	Sample Sample
	Tags   Tags
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
		Name:   name,
		Value:  value,
		Type:   MetricType(typ),
		Sample: Sample(sampleRate),
	}

	if tags := strings.Split(tags, ","); len(tags) != 0 {
		m.Tags = make(Tags, 0, len(tags))

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

func (m Metric) Format(f fmt.State, _ rune) {
	fmt.Fprintf(f, "%s:%g|%s%v%v\n", m.Name, m.Value, m.Type, m.Sample, m.Tags)
}

type Tags stats.Tags

func (tags Tags) Format(f fmt.State, _ rune) {
	if len(tags) != 0 {
		io.WriteString(f, "|#")

		for i, t := range tags {
			if i != 0 {
				io.WriteString(f, ",")
			}
			io.WriteString(f, sanitize(t.Name))
			io.WriteString(f, ":")
			io.WriteString(f, sanitize(t.Value))
		}
	}
}

type Sample float64

func (r Sample) Format(f fmt.State, _ rune) {
	if r != 0 && r != 1 {
		fmt.Fprintf(f, "|@%g", float64(r))
	}
}
