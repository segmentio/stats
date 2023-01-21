package httpstats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertoforce/stats"
)

// TestRequestContextTagPropegation verifies that the root ancestor tags are
// updated in the event the context or request has children.  It's nearly
// identical to the context_test in the stats package itself, but we want to
// keep this to ensure that changes to the request context code doesn't drift
// and cause bugs.
func TestRequestContextTagPropegation(t *testing.T) {
	// dummy request
	x := httptest.NewRequest(http.MethodGet, "http://example.com/blah", nil)

	y := RequestWithTags(x)
	assert.Equal(t, 0, len(RequestTags(y)), "Initialize request tags context value")
	RequestWithTags(y, stats.T("asdf", "qwer"))
	assert.Equal(t, 1, len(RequestTags(y)), "Adding tags should result new tags")
	assert.Equal(t, 0, len(RequestTags(x)), "Original request should have no tags (because no context with key)")

	// create a child request which creates a child context
	z := y.WithContext(context.WithValue(y.Context(), interface{}("not"), "important"))
	assert.Equal(t, 1, len(RequestTags(z)), "We should still be able to see original tags")

	// Add tags to the child context's reference to the original tag slice
	RequestWithTags(z, stats.T("zxcv", "uiop"))
	assert.Equal(t, 2, len(RequestTags(z)), "Updating tags should update local reference")
	assert.Equal(t, 2, len(RequestTags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(RequestTags(x)), "Updating tags should not appear on original request")

	RequestWithTags(z, stats.T("a", "k"), stats.T("b", "k"), stats.T("c", "k"), stats.T("d", "k"), stats.T("e", "k"), stats.T("f", "k"))
	assert.Equal(t, 8, len(RequestTags(z)), "Updating tags should update local reference")
	assert.Equal(t, 8, len(RequestTags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(RequestTags(x)), "Updating tags should not appear on original request")
}
