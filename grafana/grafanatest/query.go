package grafanatest

import (
	"time"

	"github.com/segmentio/stats/grafana"
)

// QueryResponse is an implementation of the grafana.QueryResponse interface
// which captures the values passed to its method calls.
type QueryResponse struct {
	// Results is a list of values which are either of type Timeserie or Table.
	Results []interface{}
}

// Timeserie satisfies the grafana.QueryResponse interface.
func (res *QueryResponse) Timeserie(target string) grafana.TimeserieWriter {
	t := &Timeserie{Target: target}
	res.Results = append(res.Results, t)
	return t
}

// Table satisfies the grafana.QueryResponse interface.
func (res *QueryResponse) Table(columns ...grafana.Column) grafana.TableWriter {
	t := &Table{Columns: append(make([]grafana.Column, 0, len(columns)), columns...)}
	res.Results = append(res.Results, t)
	return t
}

// Timeserie values are used by a QueryResponse to capture responses to
// timeserie queries.
type Timeserie struct {
	Target string
	Values []float64
	Times  []time.Time
}

// WriteDatapoint satisfies the grafana.TimeserieWriter interface.
func (t *Timeserie) WriteDatapoint(value float64, time time.Time) {
	t.Values = append(t.Values, value)
	t.Times = append(t.Times, time)
}

// Table values are used by a QueryResponse to capture responses to table
// queries.
type Table struct {
	Columns []grafana.Column
	Rows    [][]interface{}
}

// WriteRows satisfies the grafana.TableWriter interface.
func (t *Table) WriteRow(values ...interface{}) {
	t.Rows = append(t.Rows,
		append(make([]interface{}, 0, len(values)), values...),
	)
}
