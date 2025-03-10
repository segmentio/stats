package stats

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTagsAreSorted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tags   []Tag
		sorted bool
	}{
		{
			tags:   nil,
			sorted: true,
		},
		{
			tags:   []Tag{{"A", ""}},
			sorted: true,
		},
		{
			tags:   []Tag{{"A", ""}, {"B", ""}, {"C", ""}},
			sorted: true,
		},
		{
			tags:   []Tag{{"C", ""}, {"A", ""}, {"B", ""}},
			sorted: false,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.tags), func(t *testing.T) {
			if sorted := TagsAreSorted(test.tags); sorted != test.sorted {
				t.Error(sorted)
			}
		})
	}
}

func TestM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    map[string]string
		expected []Tag
	}{
		{
			input: map[string]string{
				"A": "",
			},
			expected: []Tag{
				T("A", ""),
			},
		},
		{
			input: map[string]string{
				"a": "A",
				"b": "B",
				"c": "C",
			},
			expected: []Tag{
				T("a", "A"),
				T("b", "B"),
				T("c", "C"),
			},
		},
	}

	for _, test := range tests {
		actual := M(test.input)
		if !reflect.DeepEqual(SortTags(test.expected), SortTags(actual)) {
			t.Errorf("expected %v, got %v", test.expected, actual)
		}
	}
}
