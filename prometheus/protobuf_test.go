package prometheus

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/encoding/protodelim"

	"github.com/segmentio/stats/v5"
)

// The Accept header Alloy sends with scrape_protocols set to
// [PrometheusProto, OpenMetricsText1.0.0, OpenMetricsText0.0.1, PrometheusText0.0.4].
const acceptProtoFirst = "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.5," +
	"application/openmetrics-text;version=1.0.0;q=0.4,application/openmetrics-text;version=0.0.1;q=0.3," +
	"text/plain;version=0.0.4;q=0.2,*/*;q=0.1"

func TestAcceptsProtobuf(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		expect bool
	}{
		{name: "none", accept: "", expect: false},
		{name: "curl", accept: "*/*", expect: false},
		{name: "protobuf first", accept: acceptProtoFirst, expect: true},
		{
			name:   "scraper defaults",
			accept: "application/openmetrics-text;version=1.0.0;q=0.5,application/openmetrics-text;version=0.0.1;q=0.4,text/plain;version=0.0.4;q=0.3,*/*;q=0.2",
			expect: false,
		},
		{
			name:   "protobuf ranked below text",
			accept: "text/plain;version=0.0.4;q=0.5,application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.4",
			expect: false,
		},
		{
			name:   "protobuf only",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited",
			expect: true,
		},
		{
			name:   "protobuf refused",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0",
			expect: false,
		},
		{
			name:   "other message",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.Other;encoding=delimited",
			expect: false,
		},
		{
			name:   "not delimited",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=text",
			expect: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if ok := acceptsProtobuf(test.accept); ok != test.expect {
				t.Errorf("acceptsProtobuf(%q) = %t, expected %t", test.accept, ok, test.expect)
			}
		})
	}
}

func scrapeProtobuf(t *testing.T, handler *Handler, gzipped bool) []*dto.MetricFamily {
	t.Helper()

	server := httptest.NewServer(handler)
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", acceptProtoFirst)
	if gzipped {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	// A Transport left to itself would negotiate and undo gzip on its own.
	res, err := (&http.Transport{DisableCompression: true}).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if ct := res.Header.Get("Content-Type"); ct != protobufContentType {
		t.Fatalf("Content-Type = %q, expected %q", ct, protobufContentType)
	}

	body := io.Reader(res.Body)
	if gzipped {
		if ce := res.Header.Get("Content-Encoding"); ce != "gzip" {
			t.Fatalf("Content-Encoding = %q, expected gzip", ce)
		}
		zr, err := gzip.NewReader(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = zr
	}

	var families []*dto.MetricFamily
	r := bufio.NewReader(body)
	for {
		mf := &dto.MetricFamily{}
		if err := protodelim.UnmarshalFrom(r, mf); err != nil {
			if errors.Is(err, io.EOF) {
				return families
			}
			t.Fatal(err)
		}
		families = append(families, mf)
	}
}

func TestServeHTTPProtobuf(t *testing.T) {
	first := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)
	later := first.Add(30 * time.Second)

	handler := &Handler{
		Buckets: map[stats.Key][]stats.Value{
			{Field: "C"}: {
				stats.ValueOf(0.25),
				stats.ValueOf(0.5),
				stats.ValueOf(0.75),
				stats.ValueOf(1.0),
			},
		},
	}

	handler.HandleMeasures(first,
		stats.Measure{Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("C", 0.1, stats.Histogram)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("B", 1, stats.Gauge)}, Tags: []stats.Tag{stats.T("a", "1"), stats.T("b", "2")}},
	)
	handler.HandleMeasures(later,
		stats.Measure{Fields: []stats.Field{stats.MakeField("A", 2, stats.Counter)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("A", 4, stats.Counter)}, Tags: []stats.Tag{stats.T("id", "123")}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("B", 42, stats.Gauge)}, Tags: []stats.Tag{stats.T("a", "1")}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("C", 0.1, stats.Histogram)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("C", 0.5, stats.Histogram)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("C", 10, stats.Histogram)}},
	)

	for _, gzipped := range []bool{false, true} {
		families := scrapeProtobuf(t, handler, gzipped)

		if len(families) != 3 {
			t.Fatalf("found %d families, expected 3: %v", len(families), families)
		}
		a, b, c := families[0], families[1], families[2]

		// Counter: the name keeps _total, and each series carries the time of
		// its own first update.
		if a.GetName() != "A_total" || a.GetType() != dto.MetricType_COUNTER || len(a.Metric) != 2 {
			t.Fatalf("bad counter family: %v", a)
		}
		if m := a.Metric[0]; len(m.Label) != 0 || m.GetCounter().GetValue() != 3 ||
			!m.GetCounter().GetCreatedTimestamp().AsTime().Equal(first) {
			t.Errorf("bad unlabeled counter: %v", m)
		}
		if m := a.Metric[1]; m.Label[0].GetName() != "id" || m.Label[0].GetValue() != "123" ||
			m.GetCounter().GetValue() != 4 || !m.GetCounter().GetCreatedTimestamp().AsTime().Equal(later) {
			t.Errorf("bad labeled counter: %v", m)
		}

		// Gauge: no created timestamp.
		if b.GetName() != "B" || b.GetType() != dto.MetricType_GAUGE || len(b.Metric) != 2 {
			t.Fatalf("bad gauge family: %v", b)
		}
		if m := b.Metric[0]; m.GetGauge().GetValue() != 42 || len(m.Label) != 1 {
			t.Errorf("bad gauge: %v", m)
		}
		if m := b.Metric[1]; m.GetGauge().GetValue() != 1 || len(m.Label) != 2 {
			t.Errorf("bad gauge: %v", m)
		}

		// Histogram: cumulative buckets without +Inf, which the scraper
		// derives from sample_count.
		if c.GetName() != "C" || c.GetType() != dto.MetricType_HISTOGRAM || len(c.Metric) != 1 {
			t.Fatalf("bad histogram family: %v", c)
		}
		h := c.Metric[0].GetHistogram()
		if h.GetSampleCount() != 4 || h.GetSampleSum() != 10.7 ||
			!h.GetCreatedTimestamp().AsTime().Equal(first) {
			t.Errorf("bad histogram: %v", h)
		}
		expect := []struct {
			le    float64
			count uint64
		}{{0.25, 2}, {0.5, 3}, {0.75, 3}, {1, 3}}
		if len(h.Bucket) != len(expect) {
			t.Fatalf("found %d buckets, expected %d: %v", len(h.Bucket), len(expect), h.Bucket)
		}
		for i, e := range expect {
			if h.Bucket[i].GetUpperBound() != e.le || h.Bucket[i].GetCumulativeCount() != e.count {
				t.Errorf("bucket %d = %v, expected le=%g count=%d", i, h.Bucket[i], e.le, e.count)
			}
		}
	}
}

// TestCreatedTimestampAfterTimeout covers a comeback: a series MetricTimeout
// deleted and that is then updated again starts from zero, so it has to carry
// a new created time or the scraper has no zero to write for it.
func TestCreatedTimestampAfterTimeout(t *testing.T) {
	first := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)
	comeback := first.Add(10 * time.Minute)

	handler := &Handler{}
	handler.HandleMeasures(first, stats.Measure{Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)}})
	handler.metrics.cleanup(first.Add(time.Minute))
	handler.HandleMeasures(comeback, stats.Measure{Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)}})

	families := scrapeProtobuf(t, handler, false)
	if len(families) != 1 || len(families[0].Metric) != 1 {
		t.Fatalf("unexpected families: %v", families)
	}
	c := families[0].Metric[0].GetCounter()
	if c.GetValue() != 1 || !c.GetCreatedTimestamp().AsTime().Equal(comeback) {
		t.Errorf("counter after timeout = %v, expected value 1 created at %s", c, comeback)
	}
}

// TestProtobufMatchesTextNames checks that the protobuf exposition renders
// family and label names the way the text format does, so a scraper that
// switches format keeps writing to the same series.
func TestProtobufMatchesTextNames(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{}
	handler.HandleMeasures(now,
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("requests.total", 1, stats.Counter)}, Tags: []stats.Tag{stats.T("status.code", "200")}},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("hits", 1, stats.Counter)}},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("hits_total", 1, stats.Counter)}},
	)

	var text strings.Builder
	handler.WriteStats(&text)

	for _, mf := range scrapeProtobuf(t, handler, false) {
		for _, m := range mf.Metric {
			series := mf.GetName()
			if len(m.Label) != 0 {
				series += "{" + m.Label[0].GetName() + "=\"" + m.Label[0].GetValue() + "\"}"
			}
			if !strings.Contains(text.String(), "\n"+series+" ") && !strings.HasPrefix(text.String(), series+" ") {
				t.Errorf("protobuf series %s not in text output:\n%s", series, text.String())
			}
		}
	}
}
