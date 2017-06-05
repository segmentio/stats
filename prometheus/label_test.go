package prometheus

import "testing"

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
