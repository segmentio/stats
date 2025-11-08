package stats

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"testing"
)

func Test_copyTags(t *testing.T) {
	t.Parallel()

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
			tags := copyTags(test.t1)
			if !reflect.DeepEqual(tags, test.t2) {
				t.Errorf("copyTags => %#v != %#v", tags, test.t2)
			}
		})
	}
}

func Test_mergeTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		t1, t2, t3 []Tag
	}{
		{
			name: "nil_inputs",
			t1:   nil,
			t2:   nil,
			t3:   nil,
		},
		{
			name: "empty_inputs",
			t1:   []Tag{},
			t2:   []Tag{},
			t3:   nil,
		},
		{
			name: "second_empty_input",
			t1:   []Tag{{"A", "1"}},
			t2:   nil,
			t3:   []Tag{{"A", "1"}},
		},
		{
			name: "first_empty_input",
			t1:   nil,
			t2:   []Tag{{"B", "2"}},
			t3:   []Tag{{"B", "2"}},
		},
		{
			name: "non_duplicated_inputs",
			t1:   []Tag{{"A", "1"}},
			t2:   []Tag{{"B", "2"}},
			t3:   []Tag{{"A", "1"}, {"B", "2"}},
		},
		{
			name: "cross_duplicated_inputs",
			t1:   []Tag{{"A", "1"}},
			t2:   []Tag{{"A", "2"}},
			t3:   []Tag{{"A", "2"}},
		},
		{
			name: "self_duplicated_input",
			t1:   []Tag{{"A", "2"}, {"A", "1"}},
			t2:   nil,
			t3:   []Tag{{"A", "1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tags := mergeTags(test.t1, test.t2)
			if !reflect.DeepEqual(tags, test.t3) {
				t.Errorf("mergeTags => %v != %v", tags, test.t3)
			}
		})
	}
}

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

func BenchmarkTagsOrder(b *testing.B) {
	b.Run("TagsAreSorted", func(b *testing.B) {
		benchmarkTagsOrder(b, TagsAreSorted)
	})
	b.Run("slices.IsSortedFunc", func(b *testing.B) {
		benchmarkTagsOrder(b, func(tags []Tag) bool {
			return slices.IsSortedFunc(tags, tagCompare)
		})
	})
	b.Run("sort.SliceIsSorted", func(b *testing.B) {
		benchmarkTagsOrder(b, func(tags []Tag) bool {
			return sort.SliceIsSorted(tags, tagIsLessByIndex(tags))
		})
	})
}

func tagIsLessByIndex(tags []Tag) func(int, int) bool {
	return func(i, j int) bool {
		return tagCompare(tags[i], tags[j]) == -1
	}
}

func benchmarkTagsOrder(b *testing.B, isSorted func([]Tag) bool) {
	b.Helper()
	b.ReportAllocs()

	tags := []Tag{
		{"A", ""},
		{"B", ""},
		{"C", ""},
		{"answer", "42"},
		{"hello", "world"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
	}

	for b.Loop() {
		isSorted(tags)
	}
}

func BenchmarkSortTags_few(b *testing.B) {
	t0 := []Tag{
		{"hello", "world"},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"A", ""},
		{"B", ""},
		{"C", ""},
	}

	benchmarkSortTags(b, t0)
}

func BenchmarkSortTags_many(b *testing.B) {
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

	benchmarkSortTags(b, t0)
}

func benchmarkSortTags(b *testing.B, t0 []Tag) {
	b.Helper()

	b.Run("SortTags", func(b *testing.B) {
		fn := func(tags []Tag) { SortTags(tags) }
		benchmarkSortTagsFunc(b, t0, fn)
	})

	b.Run("slices.SortFunc", func(b *testing.B) {
		fn := func(tags []Tag) { slices.SortFunc(tags, tagCompare) }
		benchmarkSortTagsFunc(b, t0, fn)
	})

	b.Run("slices.SortStableFunc", func(b *testing.B) {
		fn := func(tags []Tag) { slices.SortStableFunc(tags, tagCompare) }
		benchmarkSortTagsFunc(b, t0, fn)
	})

	b.Run("sort.Slice", func(b *testing.B) {
		fn := func(tags []Tag) { sort.Slice(tags, tagIsLessByIndex(tags)) }
		benchmarkSortTagsFunc(b, t0, fn)
	})

	b.Run("sort.SliceStable", func(b *testing.B) {
		fn := func(tags []Tag) { sort.SliceStable(tags, tagIsLessByIndex(tags)) }
		benchmarkSortTagsFunc(b, t0, fn)
	})
}

func benchmarkSortTagsFunc(b *testing.B, t0 []Tag, fn func([]Tag)) {
	b.Helper()
	b.ReportAllocs()

	t1 := make([]Tag, len(t0))

	for b.Loop() {
		copy(t1, t0)
		fn(t1)
	}
}

func BenchmarkTagsBufferSortSorted(b *testing.B) {
	b.ReportAllocs()

	tags := []Tag{
		{"A", ""},
		{"B", ""},
		{"C", ""},
		{"answer", "42"},
		{"answer", "42"},
		{"hello", "world"},
		{"hello", "world"},
		{"some long tag name", "!"},
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
	}

	buf := tagsBuffer{
		tags: make([]Tag, len(tags)),
	}

	for b.Loop() {
		copy(buf.tags, tags)
		buf.sort()
	}
}

func BenchmarkTagsBufferSortUnsorted(b *testing.B) {
	b.ReportAllocs()

	tags := []Tag{
		{"some long tag name", "!"},
		{"some longer tag name", "1234"},
		{"hello", "world"},
		{"C", ""},
		{"answer", "42"},
		{"hello", "world"},
		{"B", ""},
		{"answer", "42"},
		{"some long tag name", "!"},
		{"A", ""},
	}

	buf := tagsBuffer{
		tags: make([]Tag, len(tags)),
	}

	for b.Loop() {
		copy(buf.tags, tags)
		buf.sort()
	}
}

func BenchmarkMergeTags(b *testing.B) {
	b.ReportAllocs()

	origT1 := []Tag{
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
	}

	origT2 := []Tag{
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

	t1 := make([]Tag, len(origT1))
	t2 := make([]Tag, len(origT2))

	for b.Loop() {
		copy(t1, origT1)
		copy(t2, origT2)

		_ = mergeTags(t1, t2)
	}
}
