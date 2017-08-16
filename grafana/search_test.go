package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchHandler(t *testing.T) {
	sr := searchRequest{
		Target: "upper_50",
	}

	b, _ := json.Marshal(sr)

	client := http.Client{}
	server := httptest.NewServer(NewSearchHandler(
		SearchHandlerFunc(func(ctx context.Context, res SearchResponse, req *SearchRequest) error {
			if req.Target != sr.Target {
				t.Error("bad 'from' time:", req.Target, "!=", sr.Target)
			}

			res.WriteTarget("upper_25")
			res.WriteTarget("upper_50")
			res.WriteTarget("upper_90")

			res.WriteTargetValue("upper_25", 1)
			res.WriteTargetValue("upper_50", 2)

			return nil
		}),
	))
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+"/search?pretty", bytes.NewReader(b))

	r, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()

	found, _ := ioutil.ReadAll(r.Body)
	expect := searchResult

	if s := string(found); s != expect {
		t.Error(s)
		t.Log(expect)
	}
}

const searchResult = `[
  "upper_25",
  "upper_50",
  "upper_90",
  {
    "target": "upper_25",
    "value": 1
  },
  {
    "target": "upper_50",
    "value": 2
  }
]`
