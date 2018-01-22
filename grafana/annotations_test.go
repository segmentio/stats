package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnnotationsHandler(t *testing.T) {
	t0 := time.Date(2017, 8, 16, 12, 34, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Minute)

	ar := annotationsRequest{}
	ar.Range.From = t0
	ar.Range.To = t1
	ar.Annotation.Name = "name"
	ar.Annotation.Datasource = "test"
	ar.Annotation.IconColor = "rgba(255, 96, 96, 1)"
	ar.Annotation.Query = "events"
	ar.Annotation.Enable = true

	b, _ := json.Marshal(ar)

	client := http.Client{}
	server := httptest.NewServer(NewAnnotationsHandler(
		AnnotationsHandlerFunc(func(ctx context.Context, res AnnotationsResponse, req *AnnotationsRequest) error {
			if !req.From.Equal(ar.Range.From) {
				t.Error("bad 'from' time:", req.From, ar.Range.From)
			}

			if !req.To.Equal(ar.Range.To) {
				t.Error("bad 'to' time:", req.To, ar.Range.To)
			}

			if req.Name != ar.Annotation.Name {
				t.Error("bad name:", req.Name, "!=", ar.Annotation.Name)
			}

			if req.Datasource != ar.Annotation.Datasource {
				t.Error("bad datasource:", req.Datasource, "!=", ar.Annotation.Datasource)
			}

			if req.IconColor != ar.Annotation.IconColor {
				t.Error("bad icon color:", req.IconColor, "!=", ar.Annotation.IconColor)
			}

			if req.Query != ar.Annotation.Query {
				t.Error("bad query:", req.Query, "!=", ar.Annotation.Query)
			}

			if req.Enable != ar.Annotation.Enable {
				t.Error("not enabled:", req.Enable, "!=", ar.Annotation.Enable)
			}

			res.WriteAnnotation(Annotation{
				Time:     t0,
				Title:    "yay!",
				Text:     "we did it!",
				Enabled:  true,
				ShowLine: true,
				Tags:     []string{"A", "B", "C"},
			})

			return nil
		}),
	))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/annotations?pretty", bytes.NewReader(b))

	r, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	found, _ := ioutil.ReadAll(r.Body)
	expect := annotationsResult

	if s := string(found); s != expect {
		t.Error(s)
		t.Log(expect)
	}
}

const annotationsResult = `[
  {
    "annotation": {
      "name": "name",
      "datasource": "test",
      "enabled": true,
      "showLine": true
    },
    "time": 1502886840000,
    "title": "yay!",
    "text": "we did it!",
    "tags": "A, B, C"
  }
]`
