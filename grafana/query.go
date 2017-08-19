package grafana

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/segmentio/objconv"
)

// QueryHandler is the handler for the /query endpoint in the
// simple-json-datasource API.
type QueryHandler interface {
	// ServeQuery is expected to reply with a list of data points for the given
	// "target" and time range (or a set of rows for table requests).
	//
	// Note: my understanding is that "target" is some kind of identifier that
	// describes some data set in the source (like a SQL query for example), but
	// it's treated as an opaque blob of data by Grafana itself.
	ServeQuery(ctx context.Context, res QueryResponse, req *QueryRequest) error
}

// QueryHandlerFunc makes it possible to use regular function types as query
// handlers.
type QueryHandlerFunc func(context.Context, QueryResponse, *QueryRequest) error

// ServeQuery calls f, satisfies the QueryHandler interface.
func (f QueryHandlerFunc) ServeQuery(ctx context.Context, res QueryResponse, req *QueryRequest) error {
	return f(ctx, res, req)
}

// QueryResponse is an interface used to respond to a search request.
type QueryResponse interface {
	// Timeserie returns a TimeserieWriter which can be used to output the
	// datapoint in response to a timeserie request.
	Timeserie(target string) TimeserieWriter

	// Table returns a TableWriter which can be used to output the rows in
	// response to a table request.
	Table(columns ...Column) TableWriter
}

// QueryRequest represents a request received on the /query endpoint.
type QueryRequest struct {
	From          time.Time
	To            time.Time
	Interval      time.Duration
	Targets       []Target
	MaxDataPoints int
}

// Target is a data structure representing the target of a query.
type Target struct {
	Query string     `json:"target"`
	RefID string     `json:"refId"`
	Type  TargetType `json:"type"`
}

// TargetType is an enumeration of the various target types supported by
// Grafana.
type TargetType string

const (
	Timeserie TargetType = "timeserie"
	Table     TargetType = "table"
)

// TimeserieWriter is an interface used to write timeserie data in response to a
// query.
type TimeserieWriter interface {
	WriteDatapoint(value float64, time time.Time)
}

// TableWriter is an interface used to write timeserie data in response to a
// query.
type TableWriter interface {
	WriteRow(values ...interface{})
}

// Column is a data structure representing a table column.
type Column struct {
	Text string     `json:"text"`
	Type ColumnType `json:"type,omitempty"`
	Sort bool       `json:"sort,omitempty"`
	Desc bool       `json:"desc,omitempty"`
}

// Col constructs a new Column value from a text and column type.
func Col(text string, colType ColumnType) Column {
	return Column{Text: text, Type: colType}
}

// AscCol constructs a ne Column value from a text a column type, which is
// configured as a sorted column in ascending order.
func AscCol(text string, colType ColumnType) Column {
	return Column{Text: text, Type: colType, Sort: true}
}

// DescCol constructs a ne Column value from a text a column type, which is
// configured as a sorted column in descending order.
func DescCol(text string, colType ColumnType) Column {
	return Column{Text: text, Type: colType, Sort: true, Desc: true}
}

// ColumnType is an enumeration of the various column types supported by
// Grafana.
type ColumnType string

const (
	Untyped ColumnType = ""
	String  ColumnType = "string"
	Time    ColumnType = "time"
	Number  ColumnType = "number"
)

// NewQueryHandler returns a new http.Handler which delegates /query API calls
// to the given query handler.
func NewQueryHandler(handler QueryHandler) http.Handler {
	return handlerFunc(func(ctx context.Context, enc *objconv.StreamEncoder, dec *objconv.Decoder) error {
		req := queryRequest{}
		res := queryResponse{enc: enc}

		if err := dec.Decode(&req); err != nil {
			return err
		}

		if err := handler.ServeQuery(ctx, &res, &QueryRequest{
			From:          req.Range.From,
			To:            req.Range.To,
			Interval:      req.Interval,
			Targets:       req.Targets,
			MaxDataPoints: req.MaxDataPoints,
		}); err != nil {
			return err
		}

		return res.close()
	})
}

// HandleQuery installs a handler on /query.
func HandleQuery(mux *http.ServeMux, prefix string, handler QueryHandler) {
	mux.Handle(path.Join("/", prefix, "query"), NewQueryHandler(handler))
}

type queryRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type queryRequest struct {
	Range         queryRange    `json:"range"`
	Interval      time.Duration `json:"interval"`
	Targets       []Target      `json:"targets"`
	MaxDataPoints int           `json:"maxDataPoints"`
}

type queryResponse struct {
	enc       *objconv.StreamEncoder
	timeserie *timeserie
	table     *table
}

func (res *queryResponse) Timeserie(target string) TimeserieWriter {
	res.flush()
	res.timeserie = &timeserie{
		Target:     target,
		Datapoints: make([]datapoint, 0, 100),
	}
	return res.timeserie
}

func (res *queryResponse) Table(columns ...Column) TableWriter {
	res.flush()
	res.table = &table{
		Columns: columns,
		Rows:    make([]row, 0, 100),
		Type:    "table",
	}
	return res.table
}

func (res *queryResponse) close() error {
	res.flush()
	return res.enc.Close()
}

func (res *queryResponse) flush() {
	if res.timeserie != nil {
		res.enc.Encode(res.timeserie)
		res.timeserie.closed = true
		res.timeserie = nil
	}

	if res.table != nil {
		res.enc.Encode(res.table)
		res.table.closed = true
		res.table = nil
	}
}

type datapoint struct {
	value float64
	time  int64
}

func (d datapoint) EncodeValue(e objconv.Encoder) error {
	i := 0
	return e.EncodeArray(2, func(e objconv.Encoder) error {
		switch i++; i {
		case 1:
			return e.Encode(d.value)
		default:
			return e.Encode(d.time)
		}
	})
}

type timeserie struct {
	Target     string      `json:"target"`
	Datapoints []datapoint `json:"datapoints"`
	closed     bool
}

func (t *timeserie) WriteDatapoint(value float64, time time.Time) {
	if t.closed {
		panic("writing to a timeserie after it was already flushed")
	}

	t.Datapoints = append(t.Datapoints, datapoint{
		value: value,
		time:  timestamp(time),
	})
}

type row []interface{}

type table struct {
	Columns []Column `json:"columns"`
	Rows    []row    `json:"rows"`
	Type    string   `json:"type"`
	closed  bool
}

func (t *table) WriteRow(values ...interface{}) {
	if t.closed {
		panic("writing to a table after it was already flushed")
	}

	if len(values) != len(t.Columns) {
		panic(fmt.Sprintf("row value count doesn't match the number of columns, expected %d values but got %d", len(t.Columns), len(values)))
	}

	row := make(row, len(values))
	copy(row, values)

	for i := range row {
		if t, ok := row[i].(time.Time); ok {
			row[i] = timestamp(t)
		}
	}

	t.Rows = append(t.Rows, row)
}

func timestamp(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
