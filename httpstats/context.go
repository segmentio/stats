package httpstats

import (
	"net/http"

	"github.com/segmentio/stats"
)

// RequestWithTags returns a shallow copy of req with its context changed with
// this provided tags so the they can be used later.  The provided ctx must be
// non-nil.  If the context value has already been initialized, it appends the
// tags to the existing reference and returns the existing Request.
func RequestWithTags(req *http.Request, tags ...stats.Tag) *http.Request {
	return req.WithContext(stats.ContextWithTags(req.Context(), tags...))
}

func RequestTags(req *http.Request) []stats.Tag {
	return stats.ContextTags(req.Context())
}
