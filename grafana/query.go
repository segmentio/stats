package grafana

import (
	"context"
	"net/http"

	"github.com/segmentio/objconv"
)

// QueryHandler is the handler for the /query endpoint in the
// simple-json-datasource API.
type QueryHandler interface {
	ServeQuery(ctx context.Context)
}

// NewQueryHandler returns a new http.Handler which delegates /query API calls
// to the given query handler.
func NewQueryHandler(handler QueryHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, res *objconv.StreamEncoder, req *objconv.Decoder) (err error) {

		return
	})
}

// HandleQuery installs a handler on /query.
func HandleQuery(mux *http.ServeMux, handler QueryHandler) {
	mux.Handle("/query", NewQueryHandler(handler))
}
