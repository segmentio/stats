package httpstats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/segmentio/stats"
	"github.com/stretchr/testify/assert"
)

func TestContextTags(t *testing.T) {
	// dummy request
	x := httptest.NewRequest(http.MethodGet, "http://example.com/blah", nil)

	y := RequestWithTags(x)
	assert.Equal(t, 0, len(Tags(y)), "Initialize request tags context value")
	RequestWithTags(y, stats.T("asdf", "qwer"))
	assert.Equal(t, 1, len(Tags(y)), "Adding tags should result new tags")
	assert.Equal(t, 0, len(Tags(x)), "Original request should have no tags (because no context with key)")

	// create a child request which creates a child context
	z := y.WithContext(context.WithValue(y.Context(), "not", "important"))
	assert.Equal(t, 1, len(Tags(z)), "We should still be able to see original tags")

	// Add tags to the child context's reference to the original tag slice
	RequestWithTags(z, stats.T("zxcv", "uiop"))
	assert.Equal(t, 2, len(Tags(z)), "Updating tags should update local reference")
	assert.Equal(t, 2, len(Tags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(Tags(x)), "Updating tags should not appear on original request")

	RequestWithTags(z, stats.T("a", "k"), stats.T("b", "k"), stats.T("c", "k"), stats.T("d", "k"), stats.T("e", "k"), stats.T("f", "k"))
	assert.Equal(t, 8, len(Tags(z)), "Updating tags should update local reference")
	assert.Equal(t, 8, len(Tags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(Tags(x)), "Updating tags should not appear on original request")
}
