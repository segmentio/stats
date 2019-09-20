package stats

import "context"

// ContextWithTags returns a new child context with the given tags.  If the
// parent context already has tags set on it, they are _not_ propegated into
// the context children.
func ContextWithTags(ctx context.Context, tags ...Tag) context.Context {
	// initialize the context reference and return a new context
	return context.WithValue(ctx, contextKeyReqTags, &tags)
}

// ContextAddTags adds the given tags to the given context, if the tags have
// been set on any of the ancestor contexts.  ContextAddTags returns true
// if tags were successfully appended to the context, and false otherwise.
//
// The proper way to set tags on a context if you don't know whether or not
// tags already exist on the context is to first call ContextAddTags, and if
// that returns false, then call ContextWithTags instead.
func ContextAddTags(ctx context.Context, tags ...Tag) bool {
	if x := getTagSlice(ctx); x != nil {
		*x = append(*x, tags...)
		return true
	}
	return false
}

// ContextTags returns a copy of the tags on the context if they exist and nil
// if they don't exist
func ContextTags(ctx context.Context) []Tag {
	if x := getTagSlice(ctx); x != nil {
		ret := make([]Tag, len(*x))
		copy(ret, *x)
		return ret
	}
	return nil
}

func getTagSlice(ctx context.Context) *[]Tag {
	if tags, ok := ctx.Value(contextKeyReqTags).(*[]Tag); ok {
		return tags
	}
	return nil
}

// tagsKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type tagsKey struct{}

// String is Stringer implementation
func (k tagsKey) String() string {
	return "stats_tags_context_key"
}

// contextKeyReqTags is contextKey for tags
var (
	contextKeyReqTags = tagsKey{}
)
