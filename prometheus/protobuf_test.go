package prometheus

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
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
			// Prometheus' acceptHeader output for DefaultScrapeProtocols. Its
			// weights start at one above the number of protocols it knows, so
			// they shift between releases: q=0.8 on main, q=0.7 on 3.0.
			name: "prometheus 3.x defaults",
			accept: "application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.8,application/openmetrics-text;version=0.0.1;q=0.7," +
				"text/plain;version=1.0.0;escaping=allow-utf-8;q=0.6,text/plain;version=0.0.4;q=0.5,*/*;q=0.4",
			expect: false,
		},
		{
			// DefaultProtoFirstScrapeProtocols, used with scrape_native_histograms
			// or created-timestamp zero ingestion.
			name: "prometheus 3.x proto first",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.8," +
				"application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.7,application/openmetrics-text;version=0.0.1;q=0.6," +
				"text/plain;version=1.0.0;escaping=allow-utf-8;q=0.5,text/plain;version=0.0.4;q=0.4,*/*;q=0.3",
			expect: true,
		},
		{
			name: "prometheus 3.0 proto first",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7," +
				"application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.6,application/openmetrics-text;version=0.0.1;q=0.5," +
				"text/plain;version=1.0.0;escaping=allow-utf-8;q=0.4,text/plain;version=0.0.4;q=0.3,*/*;q=0.2",
			expect: true,
		},
		{
			name:   "protobuf ranked below text",
			accept: "text/plain;version=0.0.4;q=0.5,application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.4",
			expect: false,
		},
		{
			name:   "openmetrics above protobuf",
			accept: "application/openmetrics-text;version=1.0.0;q=0.5,application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.4,text/plain;version=0.0.4;q=0.3",
			expect: true,
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
	if vary := res.Header.Values("Vary"); !slices.Equal(vary, []string{"Accept", "Accept-Encoding"}) {
		t.Errorf("Vary = %q, expected Accept and Accept-Encoding", vary)
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

// TestCreatedTimestampZeroTime covers a first update with a zero time. A zero
// created time means none is set, so the series has to take the time it was
// first seen instead, or the next update would set created after the series
// already held a value and the scraper would write a zero under it.
func TestCreatedTimestampZeroTime(t *testing.T) {
	handler := &Handler{}

	before := time.Now()
	handler.HandleMeasures(time.Time{}, stats.Measure{Fields: []stats.Field{stats.MakeField("A", 5, stats.Counter)}})
	after := time.Now()
	handler.HandleMeasures(after.Add(time.Minute), stats.Measure{Fields: []stats.Field{stats.MakeField("A", 1, stats.Counter)}})

	families := scrapeProtobuf(t, handler, false)
	if len(families) != 1 || len(families[0].Metric) != 1 {
		t.Fatalf("unexpected families: %v", families)
	}
	c := families[0].Metric[0].GetCounter()
	created := c.GetCreatedTimestamp().AsTime()
	if c.GetValue() != 6 || created.Before(before) || created.After(after) {
		t.Errorf("counter = %v, expected value 6 created between %s and %s", c, before, after)
	}
}

// TestZeroTimeUpdateKeepsZeroTime checks that the created time fallback
// leaves the update time alone: MetricTimeout expiry reads the update time,
// so a zero-time first update has to expire like any other zero-time update.
func TestZeroTimeUpdateKeepsZeroTime(t *testing.T) {
	var state metricState
	state.update(counter, 5, time.Time{}, nil)

	if state.created.IsZero() {
		t.Error("created time not set by a zero-time first update")
	}
	if !state.time.IsZero() {
		t.Errorf("update time = %s, expected zero", state.time)
	}
}

// TestHistogramInfBucket checks that le="+Inf" is _count in both formats,
// where the bucket total falls short of it: a NaN observation lands in no
// bucket, and a rebuild zeroes the buckets but not _count.
func TestHistogramInfBucket(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)
	key := stats.Key{Field: "H"}
	observe := func(h *Handler, kind string, v float64) {
		h.HandleMeasures(now, stats.Measure{
			Fields: []stats.Field{stats.MakeField("H", v, stats.Histogram)},
			Tags:   []stats.Tag{stats.T("kind", kind)},
		})
	}

	handler := &Handler{Buckets: map[stats.Key][]stats.Value{key: {stats.ValueOf(1)}}}
	observe(handler, "nan", 0.5)
	observe(handler, "nan", 2)
	observe(handler, "nan", math.NaN())
	observe(handler, "rebuild", 0.5)
	observe(handler, "rebuild", 2)
	handler.Buckets[key] = []stats.Value{stats.ValueOf(1), stats.ValueOf(5)}
	observe(handler, "rebuild", 0.5)

	var text strings.Builder
	handler.WriteStats(&text)
	fromText := textSeries(t, text.String())
	fromProto := protoSeriesSet(scrapeProtobuf(t, handler, false))

	for kind, count := range map[string]float64{"nan": 3, "rebuild": 3} {
		inf := seriesKey("H_bucket", map[string]string{"kind": kind, "le": "+Inf"})
		if v := fromText[inf]; v != count {
			t.Errorf("text %s = %g, expected %g", inf, v, count)
		}
		if v := fromProto[inf]; v != count {
			t.Errorf("protobuf %s = %g, expected %g", inf, v, count)
		}
	}
	if t.Failed() {
		t.Log("text output:\n" + text.String())
	}
}

// TestProtobufConcurrentScrape scrapes while measures arrive and series
// expire, for the race detector to check the copy taken under the locks.
func TestProtobufConcurrentScrape(t *testing.T) {
	handler := &Handler{}
	start := time.Now()
	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			now := start.Add(time.Duration(i) * time.Millisecond)
			tags := []stats.Tag{stats.T("id", strconv.Itoa(i%50))}
			handler.HandleMeasures(now,
				stats.Measure{Fields: []stats.Field{stats.MakeField("c", 1, stats.Counter)}, Tags: tags},
				stats.Measure{Fields: []stats.Field{stats.MakeField("g", float64(i), stats.Gauge)}, Tags: tags},
				stats.Measure{Fields: []stats.Field{stats.MakeField("h", 0.1, stats.Histogram)}, Tags: tags},
			)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			handler.metrics.cleanup(start.Add(time.Duration(i) * time.Millisecond))
		}
	}()

	for range 50 {
		if err := handler.writeProtobuf(io.Discard); err != nil {
			t.Error(err)
		}
	}
	close(stop)
	wg.Wait()
}

// TestProtobufMatchesText checks that a scraper ingests the same series from
// the protobuf exposition as from the text format: the same names, label sets
// and values, so a scraper that switches format keeps writing to the same
// series.
//
// le is compared as a number. The scraper formats a protobuf bound itself, as
// le="1.0" where the text format writes le="1", and Prometheus 3.x normalizes
// the text value to the same form.
func TestProtobufMatchesText(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{
		Buckets: map[stats.Key][]stats.Value{
			{Measure: "http.server", Field: "duration"}: {
				stats.ValueOf(0.25),
				stats.ValueOf(1),
				stats.ValueOf(5),
			},
		},
	}
	tags := []stats.Tag{stats.T("method", "GET"), stats.T("status.code", "200")}
	handler.HandleMeasures(now,
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("requests.total", 3, stats.Counter)}, Tags: tags},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("requests.total", 1, stats.Counter)}, Tags: []stats.Tag{stats.T("method", "POST")}},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("hits", 1, stats.Counter)}},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("hits_total", 2, stats.Counter)}},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("inflight", 7, stats.Gauge)}, Tags: tags},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("duration", 0.1, stats.Histogram)}, Tags: tags},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("duration", 2, stats.Histogram)}, Tags: tags},
		stats.Measure{Name: "http.server", Fields: []stats.Field{stats.MakeField("duration", 10, stats.Histogram)}, Tags: tags},
	)

	var text strings.Builder
	handler.WriteStats(&text)
	fromText := textSeries(t, text.String())
	fromProto := protoSeriesSet(scrapeProtobuf(t, handler, false))

	for series, value := range fromText {
		if v, ok := fromProto[series]; !ok {
			t.Errorf("text series %s missing from protobuf", series)
		} else if v != value {
			t.Errorf("series %s = %g in protobuf, %g in text", series, v, value)
		}
	}
	for series := range fromProto {
		if _, ok := fromText[series]; !ok {
			t.Errorf("protobuf series %s missing from text", series)
		}
	}
	if t.Failed() {
		t.Log("text output:\n" + text.String())
	}
}

// TestProtobufCollisionOrder covers entries that render to the same name. A
// scraper keeps the first of two samples with the same name and labels, so
// the order has to be the same on every scrape, and the same as the text
// format's, or the stored value alternates between the two series.
func TestProtobufCollisionOrder(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{}
	handler.HandleMeasures(now,
		// Both render to a_b_total.
		stats.Measure{Fields: []stats.Field{stats.MakeField("a.b", 1, stats.Counter)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("a_b", 2, stats.Counter)}},
		// A counter and a gauge that both render to c_d_total.
		stats.Measure{Fields: []stats.Field{stats.MakeField("c_d", 3, stats.Counter)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("c.d_total", 4, stats.Gauge)}},
	)

	var first bytes.Buffer
	if err := handler.writeProtobuf(&first); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		var b bytes.Buffer
		if err := handler.writeProtobuf(&b); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b.Bytes(), first.Bytes()) {
			t.Fatal("protobuf exposition changed between scrapes")
		}
	}

	// The text format lists a.b first, and the c.d_total gauge before the
	// c_d counter, so a scraper keeps the same samples from either format.
	families := scrapeProtobuf(t, handler, false)
	if len(families) != 3 {
		t.Fatalf("found %d families, expected 3: %v", len(families), families)
	}
	if m := families[0].Metric; families[0].GetName() != "a_b_total" || len(m) != 2 ||
		m[0].GetCounter().GetValue() != 1 || m[1].GetCounter().GetValue() != 2 {
		t.Errorf("bad a_b_total family: %v", families[0])
	}
	if families[1].GetName() != "c_d_total" || families[1].GetType() != dto.MetricType_GAUGE ||
		families[2].GetName() != "c_d_total" || families[2].GetType() != dto.MetricType_COUNTER {
		t.Errorf("bad c_d_total families: %v %v", families[1], families[2])
	}
}

// TestHistogramCollisionOrder covers a gauge that renders to the same name
// as one of a histogram's series. Both formats list the histogram first, so
// a scraper keeps its y_count on every scrape, whichever format it reads.
func TestHistogramCollisionOrder(t *testing.T) {
	now := time.Date(2017, 6, 4, 22, 12, 0, 0, time.UTC)

	handler := &Handler{}
	handler.HandleMeasures(now,
		stats.Measure{Fields: []stats.Field{stats.MakeField("y_count", 70, stats.Gauge)}},
		stats.Measure{Fields: []stats.Field{stats.MakeField("y", 0.5, stats.Histogram)}},
	)

	var first bytes.Buffer
	handler.WriteStats(&first)
	for range 100 {
		var b bytes.Buffer
		handler.WriteStats(&b)
		if !bytes.Equal(b.Bytes(), first.Bytes()) {
			t.Fatalf("text exposition changed between scrapes:\n%s\nthen:\n%s", first.String(), b.String())
		}
	}

	var firstCount string
	for line := range strings.SplitSeq(first.String(), "\n") {
		if strings.HasPrefix(line, "y_count ") {
			firstCount = line
			break
		}
	}
	if firstCount != "y_count 1" {
		t.Errorf("text lists %q first, expected the histogram's y_count 1:\n%s", firstCount, first.String())
	}

	families := scrapeProtobuf(t, handler, false)
	if len(families) != 2 ||
		families[0].GetName() != "y" || families[0].GetType() != dto.MetricType_HISTOGRAM ||
		families[1].GetName() != "y_count" || families[1].GetType() != dto.MetricType_GAUGE {
		t.Errorf("protobuf does not list the histogram first: %v", families)
	}
}

// seriesKey renders a series as name{label="value",...} with the labels
// sorted and le normalized, so the same series is the same string whichever
// format it came from.
func seriesKey(name string, labels map[string]string) string {
	names := make([]string, 0, len(labels))
	for n := range labels {
		names = append(names, n)
	}
	slices.Sort(names)

	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i, n := range names {
		if i != 0 {
			b.WriteByte(',')
		}
		v := labels[n]
		if n == "le" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				v = strconv.FormatFloat(f, 'g', -1, 64)
			}
		}
		b.WriteString(n + "=" + strconv.Quote(v))
	}
	b.WriteByte('}')
	return b.String()
}

// textSeries parses the text exposition into its series and values.
func textSeries(t *testing.T, text string) map[string]float64 {
	t.Helper()
	set := make(map[string]float64)

	for line := range strings.SplitSeq(text, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sep := strings.LastIndexByte(line, ' ')
		value, err := strconv.ParseFloat(line[sep+1:], 64)
		if err != nil {
			t.Fatalf("bad value in %q: %v", line, err)
		}
		series := line[:sep]

		name, labels := series, map[string]string{}
		if i := strings.IndexByte(series, '{'); i >= 0 {
			name = series[:i]
			rest := strings.TrimSuffix(series[i+1:], "}")
			for rest != "" {
				eq := strings.IndexByte(rest, '=')
				v, err := strconv.QuotedPrefix(rest[eq+1:])
				if err != nil {
					t.Fatalf("bad label in %q: %v", line, err)
				}
				labels[rest[:eq]], _ = strconv.Unquote(v)
				rest = strings.TrimPrefix(rest[eq+1+len(v):], ",")
			}
		}
		set[seriesKey(name, labels)] = value
	}
	return set
}

// protoSeriesSet expands decoded families into the series a scraper stores
// for them: one per counter and gauge, and _bucket (with +Inf from
// sample_count), _sum and _count per histogram.
func protoSeriesSet(families []*dto.MetricFamily) map[string]float64 {
	set := make(map[string]float64)

	for _, mf := range families {
		for _, m := range mf.Metric {
			labels := func(extra ...string) map[string]string {
				l := make(map[string]string, len(m.Label)+1)
				for _, p := range m.Label {
					l[p.GetName()] = p.GetValue()
				}
				if len(extra) == 2 {
					l[extra[0]] = extra[1]
				}
				return l
			}

			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				set[seriesKey(mf.GetName(), labels())] = m.GetCounter().GetValue()
			case dto.MetricType_GAUGE:
				set[seriesKey(mf.GetName(), labels())] = m.GetGauge().GetValue()
			case dto.MetricType_HISTOGRAM:
				h := m.GetHistogram()
				for _, b := range h.Bucket {
					le := strconv.FormatFloat(b.GetUpperBound(), 'g', -1, 64)
					set[seriesKey(mf.GetName()+"_bucket", labels("le", le))] = float64(b.GetCumulativeCount())
				}
				set[seriesKey(mf.GetName()+"_bucket", labels("le", "+Inf"))] = float64(h.GetSampleCount())
				set[seriesKey(mf.GetName()+"_sum", labels())] = h.GetSampleSum()
				set[seriesKey(mf.GetName()+"_count", labels())] = float64(h.GetSampleCount())
			}
		}
	}
	return set
}

// benchmarkHandler returns a handler holding n series: counters with three
// labels, and one histogram series for every ten of them.
func benchmarkHandler(n int) *Handler {
	handler := &Handler{}
	now := time.Now()
	for i := range n {
		tags := []stats.Tag{
			stats.T("method", "GET"),
			stats.T("status_code", strconv.Itoa(200+i%5)),
			stats.T("url", "/v1/workspaces/"+strconv.Itoa(i)),
		}
		handler.HandleMeasures(now, stats.Measure{
			Name:   "http",
			Fields: []stats.Field{stats.MakeField("requests", 1, stats.Counter)},
			Tags:   tags,
		})
		if i%10 == 0 {
			handler.HandleMeasures(now, stats.Measure{
				Name:   "http",
				Fields: []stats.Field{stats.MakeField("duration", 0.1, stats.Histogram)},
				Tags:   tags,
			})
		}
	}
	return handler
}

// benchmarkHandlerFamilies is benchmarkHandler with its counters spread
// across families/2 names and its histograms across the other half.
func benchmarkHandlerFamilies(n, families int) *Handler {
	handler := &Handler{}
	now := time.Now()
	names := max(families/2, 1)
	for i := range n {
		tags := []stats.Tag{
			stats.T("method", "GET"),
			stats.T("status_code", strconv.Itoa(200+i%5)),
			stats.T("url", "/v1/workspaces/"+strconv.Itoa(i)),
		}
		suffix := strconv.Itoa(i % names)
		handler.HandleMeasures(now, stats.Measure{
			Name:   "http",
			Fields: []stats.Field{stats.MakeField("requests"+suffix, 1, stats.Counter)},
			Tags:   tags,
		})
		if i%10 == 0 {
			handler.HandleMeasures(now, stats.Measure{
				Name:   "http",
				Fields: []stats.Field{stats.MakeField("duration"+suffix, 0.1, stats.Histogram)},
				Tags:   tags,
			})
		}
	}
	return handler
}

// BenchmarkCollect is the text path's time under the store lock.
func BenchmarkCollect(b *testing.B) {
	handler := benchmarkHandler(100000)
	b.ReportAllocs()
	for b.Loop() {
		handler.metrics.collect(make([]metric, 0, 10000))
	}
}

// BenchmarkWriteProtobuf is the whole protobuf exposition, built and
// marshaled. peak-heap-B is the most heap in use at any write, above what
// it was before the call: the messages built but not yet written.
func BenchmarkWriteProtobuf(b *testing.B) {
	for _, families := range []int{2, 100} {
		b.Run(strconv.Itoa(families)+"-families", func(b *testing.B) {
			handler := benchmarkHandlerFamilies(100000, families)
			w := &peakHeapWriter{}
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				w.start()
				b.StartTimer()
				if err := handler.writeProtobuf(w); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(w.max), "peak-heap-B")
		})
	}
}

// peakHeapWriter discards what it is given and records the heap in use at
// each write.
type peakHeapWriter struct {
	base, max uint64
}

func (w *peakHeapWriter) start() {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	w.base = m.HeapAlloc
}

func (w *peakHeapWriter) Write(p []byte) (int, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.HeapAlloc > w.base && m.HeapAlloc-w.base > w.max {
		w.max = m.HeapAlloc - w.base
	}
	return len(p), nil
}

// BenchmarkProtoEntries is the protobuf path's time under the store lock.
func BenchmarkProtoEntries(b *testing.B) {
	handler := benchmarkHandler(100000)
	b.ReportAllocs()
	for b.Loop() {
		handler.metrics.protoEntries()
	}
}
