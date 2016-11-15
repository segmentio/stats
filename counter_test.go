package stats

import (
	"reflect"
	"testing"
)

func TestMakeCounter(t *testing.T) {
	tests := []struct {
		key  string
		name string
		tags []Tag
	}{
		{
			key:  "?",
			name: "",
			tags: nil,
		},
		{
			key:  "M?",
			name: "M",
			tags: nil,
		},
		{
			key:  "M?A=1",
			name: "M",
			tags: []Tag{{"A", "1"}},
		},
		{
			key:  "M?B=2&A=1",
			name: "M",
			tags: []Tag{{"B", "2"}, {"A", "1"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			counter := makeCounter(test.name, test.tags, nil)

			if counter.key != test.key {
				t.Errorf("makeCounter(%#v, %#v, nil) => %#v != %#v (bad key)", test.name, test.tags, counter.key, test.key)
			}

			if counter.Name() != test.name {
				t.Errorf("makeCounter(%#v, %#v, nil) => %#v != %#v (bad name)", test.name, test.tags, counter.name, test.name)
			}

			if !reflect.DeepEqual(counter.Tags(), test.tags) {
				t.Errorf("makeCounter(%#v, %#v, nil) => %#v != %#v (bad tags)", test.name, test.tags, counter.tags, test.tags)
			}
		})
	}
}

func TestCounterIncr(t *testing.T) {
	metrics := make(chan Metric, 1)
	counter := makeCounter("M", []Tag{{"A", "1"}, {"B", "2"}}, metrics)

	counter.Incr()

	if m := <-metrics; !reflect.DeepEqual(m, Metric{
		Type:  CounterType,
		Key:   "M?A=1&B=2",
		Name:  "M",
		Tags:  []Tag{{"A", "1"}, {"B", "2"}},
		Value: 1,
	}) {
		t.Errorf("counter.Incr() => %#v (bad metric)", m)
	}
}
