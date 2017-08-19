package grafanatest

import "github.com/segmentio/stats/grafana"

// AnnotationsResponse is an implementation of the grafana.AnnotationsResponse
// interface which captures the values passed to its method calls.
type AnnotationsResponse struct {
	Annotations []grafana.Annotation
}

// WriteAnnotation satisfies the grafana.AnnotationsResponse interface.
func (res *AnnotationsResponse) WriteAnnotation(a grafana.Annotation) {
	res.Annotations = append(res.Annotations, a)
}
