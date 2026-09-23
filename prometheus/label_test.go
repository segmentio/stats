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

func TestLabelLessLE(t *testing.T) {
	tests := []struct {
		name string
		l1   label
		l2   label
		less bool
	}{
		// +Inf must sort after every numeric boundary. As raw strings it
		// sorts first, because '+' is ASCII 43 and digits start at 48.
		{name: "+Inf after a boundary", l1: label{"le", "0.25"}, l2: label{"le", "+Inf"}, less: true},
		{name: "+Inf not before a boundary", l1: label{"le", "+Inf"}, l2: label{"le", "0.25"}, less: false},
		{name: "+Inf equals itself", l1: label{"le", "+Inf"}, l2: label{"le", "+Inf"}, less: false},

		// Numeric order, not lexical: "10" < "2" as a string.
		{name: "10 after 2", l1: label{"le", "10"}, l2: label{"le", "2"}, less: false},
		{name: "2 before 10", l1: label{"le", "2"}, l2: label{"le", "10"}, less: true},
		{name: "fractional", l1: label{"le", "0.005"}, l2: label{"le", "0.01"}, less: true},

		// Other labels keep comparing as strings, so a numeric-looking value
		// on a non-le label is unaffected.
		{name: "non-le stays lexical", l1: label{"code", "10"}, l2: label{"code", "2"}, less: true},

		// A non-numeric le value falls back to string comparison rather than
		// treating the parse failure as equality.
		{name: "unparseable le", l1: label{"le", "abc"}, l2: label{"le", "abd"}, less: true},

		// Name comparison still wins over value comparison.
		{name: "different names", l1: label{"a", "9"}, l2: label{"le", "1"}, less: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if less := test.l1.less(test.l2); less != test.less {
				t.Errorf("(%v < %v) = %t, expected %t", test.l1, test.l2, less, test.less)
			}
		})
	}
}
