package stats

import (
	"reflect"
	"testing"
)

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
				t.Errorf("sortTags(%#v) => %#v != %#v", test.t1, test.t2)
			}
		})
	}
}
