package grafana

import (
	"context"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/segmentio/objconv"
)

// AnnotationsHandler is the handler for the /annotations endpoint in the
// simple-json-datasource API.
type AnnotationsHandler interface {
	// ServeAnnotations
	ServeAnnotations(ctx context.Context, res AnnotationsResponse, req *AnnotationsRequest) error
}

// AnnotationsHandlerFunc makes it possible to use regular function types as annotations
// handlers.
type AnnotationsHandlerFunc func(context.Context, AnnotationsResponse, *AnnotationsRequest) error

// ServeAnnotations calls f, satisfies the AnnotationsHandler interface.
func (f AnnotationsHandlerFunc) ServeAnnotations(ctx context.Context, res AnnotationsResponse, req *AnnotationsRequest) error {
	return f(ctx, res, req)
}

// AnnotationsResponse is an interface used to an annotations request.
type AnnotationsResponse interface {
	// WriteAnnotation writes an annotation to the response. The method may be
	// called multiple times.
	WriteAnnotation(Annotation)
}

// AnnotationsRequest represents a request received on the /annotations
// endpoint.
//
// Note: It's not really clear if this request is intended to add a new
// annotation into the data source, neither how the name and datasource fields
// are supposed to be used. It seems to work to treat it as a read-only request
// for annotations on the given time range and query.
type AnnotationsRequest struct {
	From       time.Time
	To         time.Time
	Name       string
	Datasource string
	IconColor  string
	Query      string
	Enable     bool
}

// Annotation represents a single Grafana annotation.
type Annotation struct {
	Time     time.Time
	Title    string
	Text     string
	Enabled  bool
	ShowLine bool
	Tags     []string
}

// NewAnnotationsHandler returns a new http.Handler which delegates /annotations API calls
// to the given annotations handler.
func NewAnnotationsHandler(handler AnnotationsHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, enc *objconv.StreamEncoder, dec *objconv.Decoder) error {
		req := annotationsRequest{}
		res := annotationsResponse{enc: enc}

		if err := dec.Decode(&req); err != nil {
			return err
		}

		res.name = req.Annotation.Name
		res.datasource = req.Annotation.Datasource

		if err := handler.ServeAnnotations(ctx, &res, &AnnotationsRequest{
			From:       req.Range.From,
			To:         req.Range.To,
			Name:       req.Annotation.Name,
			Datasource: req.Annotation.Datasource,
			IconColor:  req.Annotation.IconColor,
			Query:      req.Annotation.Query,
			Enable:     req.Annotation.Enable,
		}); err != nil {
			return err
		}

		return enc.Close()
	})
}

// HandleAnnotations installs a handler on /annotations.
func HandleAnnotations(mux *http.ServeMux, prefix string, handler AnnotationsHandler) {
	mux.Handle(path.Join("/", prefix, "annotations"), NewAnnotationsHandler(handler))
}

type annotationsRequest struct {
	Range struct {
		From time.Time `json:"from"`
		To   time.Time `json:"to"`
	} `json:"range"`

	Annotation struct {
		Name       string `json:"name"`
		Datasource string `json:"datasource"`
		IconColor  string `json:"iconColor"`
		Query      string `json:"query"`
		Enable     bool   `json:"enable"`
	} `json:"annotation"`
}

type annotationsResponse struct {
	enc        *objconv.StreamEncoder
	name       string
	datasource string
}

func (res *annotationsResponse) WriteAnnotation(a Annotation) {
	res.enc.Encode(annotationInfo{
		Annotation: annotation{
			Name:       res.name,
			Datasource: res.datasource,
			Enabled:    a.Enabled,
			ShowLine:   a.ShowLine,
		},
		Time:  timestamp(a.Time),
		Title: a.Title,
		Text:  a.Text,
		Tags:  strings.Join(a.Tags, ", "),
	})
}

type annotationInfo struct {
	Annotation annotation `json:"annotation"`
	Time       int64      `json:"time"`
	Title      string     `json:"title"`
	Text       string     `json:"text,omitempty"`
	Tags       string     `json:"tags,omitempty"`
}

type annotation struct {
	Name       string `json:"name"`
	Datasource string `json:"datasource"`
	Enabled    bool   `json:"enabled"`
	ShowLine   bool   `json:"showLine"`
}
