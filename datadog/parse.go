package datadog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/segmentio/stats"
)

func parseMetric(s string) (m Metric, err error) {
	var next = strings.TrimSpace(s)
	var name string
	var val string
	var typ string
	var rate string
	var tags string

	val, next = nextToken(next, '|')
	typ, next = nextToken(next, '|')
	rate, tags = nextToken(next, '|')
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
		Type:  MetricType(typ),
		Name:  name,
		Value: value,
		Rate:  sampleRate,
	}

	if len(tags) != 0 {
		m.Tags = make([]stats.Tag, 0, count(tags, ',')+1)

		for len(tags) != 0 {
			var tag string

			if tag, tags = nextToken(tags, ','); len(tag) != 0 {
				name, value := split(tag, ':')
				m.Tags = append(m.Tags, stats.Tag{name, value})
			}
		}
	}

	return
}

func nextToken(s string, b byte) (token string, next string) {
	if off := strings.IndexByte(s, b); off >= 0 {
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

func count(s string, b byte) (n int) {
	for {
		if off := strings.IndexByte(s, b); off < 0 {
			break
		} else {
			n++
			s = s[off+1:]
		}
	}
	return
}
