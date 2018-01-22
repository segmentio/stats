package grafanatest

// SearchResponse is an implementation of the grafana.SearchResponse interface
// which captures the values passed to its method calls.
type SearchResponse struct {
	Targets []string
	Values  []interface{}
}

// WriteTarget satisfies the grafana.SearchResponse interface.
func (res *SearchResponse) WriteTarget(target string) {
	res.WriteTargetValue(target, nil)
}

// WriteTargetValue satisfies the grafana.SearchResponse interface.
func (res *SearchResponse) WriteTargetValue(target string, value interface{}) {
	res.Targets = append(res.Targets, target)
	res.Values = append(res.Values, value)
}
