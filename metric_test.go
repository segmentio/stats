package stats

import "testing"

func TestMetricKey(t *testing.T) {
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
			key:  "M?A=1&B=2",
			name: "M",
			tags: []Tag{{"A", "1"}, {"B", "2"}},
		},
		{
			key:  "M?A=1&B=2&C=3",
			name: "M",
			tags: []Tag{{"A", "1"}, {"B", "2"}, {"C", "3"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if key := metricKey(test.name, test.tags); key != test.key {
				t.Errorf("metricKey(%#v, %#v) => %#v != %#v", test.name, test.tags, key, test.key)
			}
		})
	}
}
