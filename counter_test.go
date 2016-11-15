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
	channel := make(chan metric, 1)
	counter := makeCounter("M", []Tag{{"A", "1"}, {"B", "2"}}, channel)

	counter.Incr()

	if m := <-channel; !reflect.DeepEqual(m, metric{
		key:   "M?A=1&B=2",
		name:  "M",
		tags:  []Tag{{"A", "1"}, {"B", "2"}},
		value: 1,
	}) {
		t.Errorf("counter.Incr() => %#v (bad metric)", m)
	}
}
