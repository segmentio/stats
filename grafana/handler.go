package grafana

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/segmentio/objconv"
	"github.com/segmentio/objconv/json"
)

// Handler is an interface that must be implemented by types that intend to act
// as a grafana data source using the simple-json-datasource plugin.
//
// See https://github.com/grafana/simple-json-datasource for more details about
// the implementation.
type Handler interface {
	AnnotationsHandler
	QueryHandler
	SearchHandler
}

// NewHandler returns a new http.Handler that implements the
// simple-json-datasource API.
func NewHandler(handler Handler) http.Handler {
	mux := http.NewServeMux()
	Handle(mux, handler)
	return mux
}

// Handle installs a handler implementing the simple-json-datasource API on mux.
//
// The function adds three routes to mux, for /annotations, /query, and /search.
func Handle(mux *http.ServeMux, handler Handler) {
	HandleAnnotations(mux, handler)
	HandleQuery(mux, handler)
	HandleSearch(mux, handler)

	// Registering a global handler is a common thing that applications do, to
	// avoid overriding one that may already exist we first check that none were
	// previously registered.
	if _, pattern := mux.Handler(&http.Request{
		URL: &url.URL{Path: "/"},
	}); len(pattern) == 0 {
		mux.Handle("/", handlerFunc(func(ctx context.Context, enc *objconv.StreamEncoder, dec *objconv.Decoder) error {
			return nil
		}))
	}
}

func handlerFunc(f func(context.Context, *objconv.StreamEncoder, *objconv.Decoder) error) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		h := res.Header()
		h.Set("Access-Control-Allow-Headers", "Accept, Content-Type")
		h.Set("Access-Control-Allow-Methods", "POST")
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set("Server", "stats/grafana (simple-json-datasource)")

		switch req.Method {
		case http.MethodPost:
		case http.MethodOptions:
			res.WriteHeader(http.StatusOK)
			return
		default:
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := f(
			req.Context(),
			newEncoder(res, req),
			newDecoder(req.Body),
		)

		// TODO: support different error types to return different error codes?
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Printf("grafana: %s %s: %s", req.Method, req.URL.Path, err)
		}
	})
}

func newEncoder(res http.ResponseWriter, req *http.Request) *objconv.StreamEncoder {
	q := req.URL.Query()
	if _, ok := q["pretty"]; ok {
		return json.NewPrettyStreamEncoder(res)
	} else {
		return json.NewStreamEncoder(res)
	}
}

func newDecoder(r io.Reader) *objconv.Decoder {
	return json.NewDecoder(r)
}
