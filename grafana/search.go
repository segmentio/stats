package grafana

import (
	"context"
	"net/http"
	"path"

	"github.com/segmentio/objconv"
)

// SearchHandler is the handler for the /search endpoint in the
// simple-json-datasource API.
type SearchHandler interface {
	// ServeSearch is expected to implement the search functionality of a
	// Grafana data source.
	//
	// Note: It's not really clear how search is implemented, I think the
	// "target" field in the request is some kind of prefix or keyword to
	// use to return a list of potential matches that can be used in a /query
	// request.
	ServeSearch(ctx context.Context, res SearchResponse, req *SearchRequest) error
}

// SearchHandlerFunc makes it possible to use regular function types as search
// handlers.
type SearchHandlerFunc func(context.Context, SearchResponse, *SearchRequest) error

// ServeSearch calls f, satisfies the SearchHandler interface.
func (f SearchHandlerFunc) ServeSearch(ctx context.Context, res SearchResponse, req *SearchRequest) error {
	return f(ctx, res, req)
}

// SearchResponse is an interface used to respond to a search request.
type SearchResponse interface {
	// WriteTarget writes target in the response, the method may be called
	// multiple times.
	WriteTarget(target string)

	// WriteTargetValue writes the pair of target and value in the response,
	// the method may be called multiple times.
	WriteTargetValue(target string, value interface{})
}

// SearhRequest represents a request received on the /search endpoint.
type SearchRequest struct {
	Target string
}

// NewSearchHandler returns a new http.Handler which delegates /search API calls
// to the given search handler.
func NewSearchHandler(handler SearchHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, enc *objconv.StreamEncoder, dec *objconv.Decoder) error {
		req := searchRequest{}
		res := searchResponse{enc: enc}

		if err := dec.Decode(&req); err != nil {
			return err
		}

		if err := handler.ServeSearch(ctx, &res, &SearchRequest{
			Target: req.Target,
		}); err != nil {
			return err
		}

		return enc.Close()
	})
}

// HandleSearch installs a handler on /search.
func HandleSearch(mux *http.ServeMux, prefix string, handler SearchHandler) {
	mux.Handle(path.Join("/", prefix, "search"), NewSearchHandler(handler))
}

type searchRequest struct {
	Target string `json:"target"`
}

type searchResponse struct {
	enc *objconv.StreamEncoder
}

func (res *searchResponse) WriteTarget(target string) {
	res.enc.Encode(target)
}

func (res *searchResponse) WriteTargetValue(target string, value interface{}) {
	res.enc.Encode(struct {
		Target string      `json:"target"`
		Value  interface{} `json:"value"`
	}{target, value})
}
