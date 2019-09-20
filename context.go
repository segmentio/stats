package stats

import "context"

func ContextWithTags(ctx context.Context, tags ...Tag) context.Context {
	if x := getTagSlice(ctx); x != nil {
		*x = append(*x, tags...)
		return ctx
	}
	// initialize the context reference and return a new context
	return context.WithValue(ctx, contextKeyReqTags, &tags)
}

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
func (k *tagsKey) String() string {
	return "stats_tags_context_key"
}

// contextKeyReqTags is contextKey for tags
var (
	contextKeyReqTags = &tagsKey{}
)
