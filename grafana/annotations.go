package grafana

import (
	"context"
	"net/http"

	"github.com/segmentio/objconv"
)

// AnnotationsHandler is the handler for the /annotations endpoint in the
// simple-json-datasource API.
type AnnotationsHandler interface {
	ServeAnnotations(ctx context.Context) error
}

// NewAnnotationsHandler returns a new http.Handler which delegates /annotations API calls
// to the given annotations handler.
func NewAnnotationsHandler(handler AnnotationsHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, res *objconv.StreamEncoder, req *objconv.Decoder) (err error) {

		return
	})
}

// HandleAnnotations installs a handler on /annotations.
func HandleAnnotations(mux *http.ServeMux, handler AnnotationsHandler) {
	mux.Handle("/annotations", NewAnnotationsHandler(handler))
}
