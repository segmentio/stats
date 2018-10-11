package stats

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestCopyTags(t *testing.T) {
	tests := []struct {
		t1 []Tag
		t2 []Tag
	}{
		{
			t1: nil,
			t2: nil,
		},
		{
			t1: []Tag{},
			t2: nil,
		},
		{
			t1: []Tag{{"A", "1"}, {"B", "2"}, {"C", "3"}},
			t2: []Tag{{"A", "1"}, {"B", "2"}, {"C", "3"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if tags := copyTags(test.t1); !reflect.DeepEqual(tags, test.t2) {
				t.Errorf("copyTags => %#v != %#v", tags, test.t2)
			}
		})
	}
}

func TestConcatTags(t *testing.T) {
	tests := []struct {
		t1 []Tag
		t2 []Tag
		t3 []Tag
	}{
		{
			t1: nil,
			t2: nil,
			t3: nil,
		},
		{
			t1: []Tag{},
			t2: []Tag{},
			t3: nil,
		},
		{
			t1: []Tag{{"A", "1"}},
			t2: nil,
			t3: []Tag{{"A", "1"}},
		},
		{
			t1: nil,
			t2: []Tag{{"B", "2"}},
			t3: []Tag{{"B", "2"}},
		},
		{
			t1: []Tag{{"A", "1"}},
			t2: []Tag{{"B", "2"}},
			t3: []Tag{{"A", "1"}, {"B", "2"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if tags := concatTags(test.t1, test.t2); !reflect.DeepEqual(tags, test.t3) {
				t.Errorf("concatTags => %#v != %#v", tags, test.t3)
			}
		})
	}
}

func TestTagsAreSorted(t *testing.T) {
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
		t.Run(fmt.Sprintf("%v", test.tags), func(t *testing.T) {
			if sorted := TagsAreSorted(test.tags); sorted != test.sorted {
				t.Error(sorted)
			}
		})
	}
}

func TestM(t *testing.T) {
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

func BenchmarkTagsOrder(b *testing.B) {
	b.Run("TagsAreSorted", func(b *testing.B) {
		benchmarkTagsOrder(b, TagsAreSorted)
	})
	b.Run("sort.IsSorted(tags)", func(b *testing.B) {
		benchmarkTagsOrder(b, func(tags []Tag) bool { return sort.IsSorted(tagsByName(tags)) })
	})
}

func benchmarkTagsOrder(b *testing.B, isSorted func([]Tag) bool) {
	tags := []Tag{
		{"A", ""},
		{"B", ""},
		{"C", ""},
		{"answer", "42"},
		{"hello", "world"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
	}

	for i := 0; i != b.N; i++ {
		isSorted(tags)
	}
}

func BenchmarkSortTags(b *testing.B) {
	t0 := []Tag{
		{"hello", "world"},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"A", ""},
		{"B", ""},
		{"C", ""},
	}

	t1 := make([]Tag, len(t0))

	for i := 0; i != b.N; i++ {
		copy(t1, t0)
		SortTags(t1)
	}
}

func BenchmarkSortTagsMany(b *testing.B) {
	t0 := []Tag{
		{"hello", "world"},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"A", ""},
		{"B", ""},
		{"C", ""},
		{"hello", "world"},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"A", ""},
		{"B", ""},
		{"C", ""},
		{"hello", "world"},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"A", ""},
		{"B", ""},
		{"C", ""},
	}

	t1 := make([]Tag, len(t0))

	for i := 0; i != b.N; i++ {
		copy(t1, t0)
		SortTags(t1)
	}
}
