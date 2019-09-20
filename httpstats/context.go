package httpstats

import (
	"context"
	"net/http"

	"github.com/segmentio/stats"
)

// RequestWithTags returns a shallow copy of req with its context changed with
// this provided tags so the they can be used later.  The provided ctx must be
// non-nil.  If the context value has already been initialized, it appends the
// tags to the existing reference and returns the existing Request.
func RequestWithTags(req *http.Request, tags ...stats.Tag) *http.Request {
	ctx := req.Context()
	// first try adding tags to the request
	if AddTagsToRequest(req, tags...) {
		return req
	}
	// otherwise initialize the context reference and return a shallow copy
	ctx = context.WithValue(ctx, contextKeyReqTags, &tags)
	return req.WithContext(ctx)
}

// AddTagsToRequest adds the given tags to the request if the reqeuest context
// has already been initialized with tags. It returns a bool which is true if
// the context value exists and they were added, false otherwise.
func AddTagsToRequest(req *http.Request, tags ...stats.Tag) bool {
	cur := getReqTagsSlice(req)
	if cur == nil {
		return false
	}
	*cur = append(*cur, tags...)
	return true
}

// Tags returns a copy of the tags on the request, if they exist.  Otherwise it
// returns nil.
func Tags(req *http.Request) []stats.Tag {
	if x := getReqTagsSlice(req); x != nil {
		ret := make([]stats.Tag, len(*x))
		copy(ret, *x)
		return ret
	}
	return nil
}

func getReqTagsSlice(req *http.Request) *[]stats.Tag {
	if tags, ok := req.Context().Value(contextKeyReqTags).(*[]stats.Tag); ok {
		return tags
	}
	return nil
}

// tagsKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type tagsKey struct{}

// String is Stringer implementation
func (k *tagsKey) String() string {
	return "stats_tags_context_key"
}

// contextKeyReqTags is contextKey for tags
var (
	contextKeyReqTags = &tagsKey{}
)
