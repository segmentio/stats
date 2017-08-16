package grafana

import (
	"context"
	"net/http"

	"github.com/segmentio/objconv"
)

// SearchHandler is the handler for the /search endpoint in the
// simple-json-datasource API.
type SearchHandler interface {
	QuerySearch(ctx context.Context) error
}

// NewSearchHandler returns a new http.Handler which delegates /search API calls
// to the given search handler.
func NewSearchHandler(handler SearchHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, res *objconv.StreamEncoder, req *objconv.Decoder) (err error) {

		return
	})
}

// HandleSearch installs a handler on /search.
func HandleSearch(mux *http.ServeMux, handler SearchHandler) {
	mux.Handle("/search", NewSearchHandler(handler))
}
