package stats

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNewMetric(t *testing.T) {
	tests := []struct {
		metric Metric
		name   string
		string string
		tags   Tags
	}{
		{
			metric: NewMetric("hello"),
			name:   "hello",
			string: `hello {}`,
			tags:   Tags{},
		},
		{
			metric: NewMetric("hello", Tag{"answer", "42"}, Tag{"hello", "world"}),
			name:   "hello",
			string: `hello {"answer":"42","hello":"world"}`,
			tags:   Tags{{"answer", "42"}, {"hello", "world"}},
		},
	}

	for _, test := range tests {
		metric := NewMetric(test.name, test.tags...)

		if name := metric.Name(); name != test.name {
			t.Errorf("%#v: invalid name: %%v != %%v", test.metric, test.name, name)
		}

		if tags := metric.Tags(); !reflect.DeepEqual(tags, test.tags) {
			t.Errorf("%#v: invalid tags: %s != %s", test.metric, test.tags, tags)
		}

		if string := fmt.Sprint(metric); string != test.string {
			t.Errorf("%#v: invalid string: %#v != %#v", test.metric, test.string, string)
		}
	}
}
