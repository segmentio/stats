package httpstats

import (
	"net/http"

	"github.com/vertoforce/stats"
)

// RequestWithTags returns a shallow copy of req with its context updated to
// include the given tags.  If the context already contains tags, then those
// are appended to, otherwise a new context containing the tags is created and
// attached to the new request
// ⚠️ : Using this may blow up the cardinality of your httpstats metrics. Use
// with care for tags with low cardinalities.
func RequestWithTags(req *http.Request, tags ...stats.Tag) *http.Request {
	if stats.ContextAddTags(req.Context(), tags...) {
		return req.WithContext(req.Context())
	}
	return req.WithContext(stats.ContextWithTags(req.Context(), tags...))
}

// RequestTags returns the tags associated with the request, if any.  It
// returns nil otherwise.
func RequestTags(req *http.Request) []stats.Tag {
	return stats.ContextTags(req.Context())
}
