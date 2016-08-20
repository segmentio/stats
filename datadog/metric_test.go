package datadog

import (
	"fmt"
	"reflect"
	"testing"
)

func TestParseSuccess(t *testing.T) {
	tests := []struct {
		s string
		m Metric
	}{
		{
			s: "page.views:1|c\n",
			m: Metric{
				Name:       "page.views",
				Value:      1,
				Type:       Counter,
				SampleRate: 1,
				Tags:       Tags{},
			},
		},

		{
			s: "fuel.level:0.5|g\n",
			m: Metric{
				Name:       "fuel.level",
				Value:      0.5,
				Type:       Gauge,
				SampleRate: 1,
				Tags:       Tags{},
			},
		},

		{
			s: "song.length:240|h|@0.5\n",
			m: Metric{
				Name:       "song.length",
				Value:      240,
				Type:       Histogram,
				SampleRate: 0.5,
				Tags:       Tags{},
			},
		},

		{
			s: "users.uniques:1234|s\n",
			m: Metric{
				Name:       "users.uniques",
				Value:      1234,
				Type:       Set,
				SampleRate: 1,
				Tags:       Tags{},
			},
		},

		{
			s: "users.online:1|c|#country:china\n",
			m: Metric{
				Name:       "users.online",
				Value:      1,
				Type:       Counter,
				SampleRate: 1,
				Tags:       Tags{{"country", "china"}},
			},
		},

		{
			s: "users.online:1|c|@0.5|#country:china\n",
			m: Metric{
				Name:       "users.online",
				Value:      1,
				Type:       Counter,
				SampleRate: 0.5,
				Tags:       Tags{{"country", "china"}},
			},
		},
	}

	for _, test := range tests {
		if m, err := ParseMetric(test.s); err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(m, test.m) {
			t.Errorf("%#v:\n- %#v\n- %#v", test.s, test.m, m)
		} else if s := fmt.Sprint(m); s != test.s {
			t.Errorf("%#v\n%#v", test.s, s)
		}
	}
}

func TestParseFailure(t *testing.T) {
	tests := []string{
		"",
		":10|c",             // missing name
		"name:|c",           // missing value
		"name:abc|c",        // malformed value
		"name:1",            // missing type
		"name:1|",           // missing type
		"name:1|c|???",      // malformed sample rate
		"name:1|c|@abc",     // malformed sample rate
		"name:1|c|@0.5|???", // malformed tags
	}

	for _, test := range tests {
		if _, err := ParseMetric(test); err == nil {
			t.Error("%#v: expected parsing error", test)
		}
	}
}
