package grafana

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

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
func NewHandler(prefix string, handler Handler) http.Handler {
	mux := http.NewServeMux()
	Handle(mux, prefix, handler)
	return mux
}

// Handle installs a handler implementing the simple-json-datasource API on mux.
//
// The function adds three routes to mux, for /annotations, /query, and /search.
func Handle(mux *http.ServeMux, prefix string, handler Handler) {
	HandleAnnotations(mux, prefix, handler)
	HandleQuery(mux, prefix, handler)
	HandleSearch(mux, prefix, handler)

	// Registering a global handler is a common thing that applications do, to
	// avoid overriding one that may already exist we first check that none were
	// previously registered.
	root := path.Join("/", prefix)

	if _, pattern := mux.Handler(&http.Request{
		URL: &url.URL{Path: root},
	}); len(pattern) == 0 {
		mux.HandleFunc(root, func(res http.ResponseWriter, req *http.Request) {
			setResponseHeaders(res)
		})
	}
}

func handlerFunc(f func(context.Context, *objconv.StreamEncoder, *objconv.Decoder) error) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		setResponseHeaders(res)

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
		switch err {
		case nil:
		case context.Canceled:
			res.WriteHeader(http.StatusInternalServerError)
		case context.DeadlineExceeded:
			res.WriteHeader(http.StatusGatewayTimeout)
		default:
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

func setResponseHeaders(res http.ResponseWriter) {
	h := res.Header()
	h.Set("Access-Control-Allow-Headers", "Accept, Content-Type")
	h.Set("Access-Control-Allow-Methods", "POST")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Server", "stats/grafana (simple-json-datasource)")
}
