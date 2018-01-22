package grafana

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/objconv/json"
)

func TestQueryHandler(t *testing.T) {
	t0 := time.Date(2017, 8, 16, 12, 34, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Minute)

	qr := queryRequest{
		Range:    queryRange{From: t0, To: t1},
		Interval: 1 * time.Second,
		Targets: []Target{
			{Query: "upper_50", RefID: "A", Type: Timeserie},
			{Query: "upper_75", RefID: "B", Type: Timeserie},
			{Query: "entries", RefID: "C", Type: Table},
		},
		MaxDataPoints: 150,
	}

	b, _ := json.Marshal(qr)

	client := http.Client{}
	server := httptest.NewServer(NewQueryHandler(
		QueryHandlerFunc(func(ctx context.Context, res QueryResponse, req *QueryRequest) error {
			if !req.From.Equal(t0) {
				t.Error("bad 'from' time:", req.From, "!=", t0)
			}

			if !req.To.Equal(t1) {
				t.Error("bad 'to' time:", req.To, "!=", t1)
			}

			if req.Interval != qr.Interval {
				t.Error("bad interval:", req.Interval, "!=", qr.Interval)
			}

			if req.MaxDataPoints != qr.MaxDataPoints {
				t.Error("bad max-data-points:", req.MaxDataPoints, "!=", qr.MaxDataPoints)
			}

			if !reflect.DeepEqual(req.Targets, qr.Targets) {
				t.Error("bad targets:")
				t.Log(req.Targets)
				t.Log(qr.Targets)
			}

			for _, target := range req.Targets {
				switch target.Type {
				case Timeserie:
					t := res.Timeserie(target.Query)
					t.WriteDatapoint(622, t0)
					t.WriteDatapoint(265, t1)

				case Table:
					t := res.Table(Col("Time", Time), Col("Country", String), Col("Number", Number))
					t.WriteRow(t0, "SE", 123)
					t.WriteRow(t0, "DE", 231)
					t.WriteRow(t1, "US", 321)
				}
			}

			return nil
		}),
	))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/query?pretty", bytes.NewReader(b))

	r, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	found, _ := ioutil.ReadAll(r.Body)
	expect := queryResult

	if s := string(found); s != expect {
		t.Error(s)
		t.Log(expect)
	}
}

const queryResult = `[
  {
    "target": "upper_50",
    "datapoints": [
      [
        622,
        1502886840000
      ],
      [
        265,
        1502886900000
      ]
    ]
  },
  {
    "target": "upper_75",
    "datapoints": [
      [
        622,
        1502886840000
      ],
      [
        265,
        1502886900000
      ]
    ]
  },
  {
    "columns": [
      {
        "text": "Time",
        "type": "time"
      },
      {
        "text": "Country",
        "type": "string"
      },
      {
        "text": "Number",
        "type": "number"
      }
    ],
    "rows": [
      [
        1502886840000,
        "SE",
        123
      ],
      [
        1502886840000,
        "DE",
        231
      ],
      [
        1502886900000,
        "US",
        321
      ]
    ],
    "type": "table"
  }
]`
