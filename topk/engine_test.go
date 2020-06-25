package topk

import (
	"testing"

	"github.com/segmentio/stats/v4"
	"github.com/stretchr/testify/require"
)

func TestTagsToString(t *testing.T) {
	for _, test := range []struct {
		name     string
		tags     []stats.Tag
		expected string
	}{
		{
			name:     "empty",
			expected: "",
		},
		{
			name:     "one tag",
			tags:     []stats.Tag{stats.T("foo", "bar")},
			expected: "foo=bar",
		},
		{
			name:     "multiple tags",
			tags:     []stats.Tag{stats.T("foo", "bar"), stats.T("bar", "baz")},
			expected: "foo=bar,bar=baz",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tagString := tagsToString(test.tags...)
			require.EqualValues(t, test.expected, tagString)
		})
	}
}
