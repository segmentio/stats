package stats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextTags(t *testing.T) {
	x := context.Background()

	y := ContextWithTags(x)
	assert.Equal(t, 0, len(ContextTags(y)), "Initialize context tags context value")
	ContextAddTags(y, T("asdf", "qwer"))
	assert.Equal(t, 1, len(ContextTags(y)), "Adding tags should result new tags")
	assert.Equal(t, 0, len(ContextTags(x)), "Original context should have no tags (because no context with key)")

	// create a child context which creates a child context
	z := context.WithValue(y, "not", "important")
	assert.Equal(t, 1, len(ContextTags(z)), "We should still be able to see original tags")

	// Add tags to the child context's reference to the original tag slice
	ContextAddTags(z, T("zxcv", "uiop"))
	assert.Equal(t, 2, len(ContextTags(z)), "Updating tags should update local reference")
	assert.Equal(t, 2, len(ContextTags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(ContextTags(x)), "Updating tags should not appear on original context")

	ContextAddTags(z, T("a", "k"), T("b", "k"), T("c", "k"), T("d", "k"), T("e", "k"), T("f", "k"))
	assert.Equal(t, 8, len(ContextTags(z)), "Updating tags should update local reference")
	assert.Equal(t, 8, len(ContextTags(y)), "Updating tags should update parent reference")
	assert.Equal(t, 0, len(ContextTags(x)), "Updating tags should not appear on original context")
}
