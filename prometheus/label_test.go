package prometheus

import (
	"reflect"
	"testing"
)

func TestLabelsLess(t *testing.T) {
	tests := []struct {
		l1   labels
		l2   labels
		less bool
	}{
		{
			l1:   labels{},
			l2:   labels{},
			less: false,
		},

		{
			l1:   labels{},
			l2:   labels{{"id", "123"}},
			less: true,
		},

		{
			l1:   labels{{"id", "123"}},
			l2:   labels{},
			less: false,
		},

		{
			l1:   labels{{"id", "123"}},
			l2:   labels{{"id", "123"}},
			less: false,
		},

		{
			l1:   labels{{"a", "1"}},
			l2:   labels{{"a", "1"}, {"b", "2"}},
			less: true,
		},

		{
			l1:   labels{{"a", "1"}, {"b", "2"}},
			l2:   labels{{"a", "1"}},
			less: false,
		},

		{
			l1:   labels{{"a", "1"}, {"b", "2"}},
			l2:   labels{{"a", "1"}, {"b", "2"}},
			less: false,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if less := test.l1.less(test.l2); less != test.less {
				t.Errorf("(%#v < %#v) != %t", test.l1, test.l2, test.less)
			}
		})
	}
}

func TestIgnoreNamed(t *testing.T) {
	var testFilter = []byte{'l', '1', 0x00, 'l', '2'}
	var testLabels = labels{
		{
			name: "l1",
		},
		{
			name: "l2",
		},
		{
			name: "l3",
		},
		{
			name: "l4",
		},
	}
	tests := []struct {
		name   string
		in     labels
		filter []byte
		expect labels
	}{
		{
			name:   "no labels",
			in:     labels(nil),
			filter: testFilter,
			expect: make(labels, 0),
		},
		{
			name:   "no ignored labels",
			in:     testLabels,
			filter: []byte(nil),
			expect: testLabels,
		},
		{
			name: "actively ignoring labels",
			in: testLabels,
			filter:testFilter,
			expect: labels{{name: "l3"}, {name: "l4"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := test.in.ignoreNamed(test.filter)

			if !reflect.DeepEqual(l, test.expect) {
				t.Errorf("\nexpected: %#v\n     got: %#v", test.expect, l)
			}
		})
	}
}
