package stats

import (
	"reflect"
	"testing"
)

func TestConcatTags(t *testing.T) {
	tests := []struct {
		t1 []Tag
		t2 []Tag
		t3 []Tag
	}{
		{
			t1: nil,
			t2: nil,
			t3: []Tag{},
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

func TestSortTags(t *testing.T) {
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
			t2: []Tag{},
		},
		{
			t1: []Tag{{"A", "1"}},
			t2: []Tag{{"A", "1"}},
		},
		{
			t1: []Tag{{"A", "1"}, {"A", "2"}},
			t2: []Tag{{"A", "1"}, {"A", "2"}},
		},
		{
			t1: []Tag{{"A", "1"}, {"B", "2"}},
			t2: []Tag{{"A", "1"}, {"B", "2"}},
		},
		{
			t1: []Tag{{"B", "2"}, {"A", "1"}},
			t2: []Tag{{"A", "1"}, {"B", "2"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			sortTags(test.t1)

			if !reflect.DeepEqual(test.t1, test.t2) {
				t.Errorf("sortTags => %#v != %#v", test.t1, test.t2)
			}
		})
	}
}
