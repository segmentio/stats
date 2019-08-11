package stats_test

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/datadog"
	"github.com/segmentio/stats/influxdb"
	"github.com/segmentio/stats/prometheus"
	"github.com/segmentio/stats/statstest"
)

func TestEngine(t *testing.T) {
	tests := []struct {
		scenario string
		function func(*testing.T, *stats.Engine)
	}{
		{
			scenario: "calling Engine.WithPrefix returns a copy of the engine with the prefix and tags inherited from the original",
			function: testEngineWithPrefix,
		},
		{
			scenario: "calling Engine.WithTags returns a copy of the engine with the prefix and tags inherited from the original",
			function: testEngineWithPrefix,
		},
		{
			scenario: "calling Engine.Flush calls Flush the handler's Flush method",
			function: testEngineFlush,
		},
		{
			scenario: "calling Engine.Incr produces a counter increment of one",
			function: testEngineIncr,
		},
		{
			scenario: "calling Engine.Add produces a counter increment of the expected amount",
			function: testEngineAdd,
		},
		{
			scenario: "calling Engine.Set produces the expected gauge value",
			function: testEngineSet,
		},
		{
			scenario: "calling Engine.Observe produces the expected histogram value",
			function: testEngineObserve,
		},
		{
			scenario: "calling Engine.Report produces the expected measures",
			function: testEngineReport,
		},
		{
			scenario: "calling Engine.Report with an array of metrics produces the expected measures",
			function: testEngineReportArray,
		},
		{
			scenario: "calling Engine.Report with a slice of metrics produces the expected measures",
			function: testEngineReportSlice,
		},
		{
			scenario: "calling Engine.Clock produces expected metrics",
			function: testEngineClock,
		},
	}

	for _, test := range tests {
		testFunc := test.function
		t.Run(test.scenario, func(t *testing.T) {
			t.Parallel()
			h := &statstest.Handler{}
			testFunc(t, stats.NewEngine("test", h, stats.T("service", "test-service")))
		})
	}
}

func testEngineWithPrefix(t *testing.T, eng *stats.Engine) {
	e2 := eng.WithPrefix("subtest", stats.T("command", "hello world"))

	if e2.Prefix != "test.subtest" {
		t.Error("bad prefix:", e2.Prefix)
	}

	if !reflect.DeepEqual(e2.Tags, []stats.Tag{
		stats.T("command", "hello world"),
		stats.T("service", "test-service"),
	}) {
		t.Error("bad tags:", e2.Tags)
	}
}

func testEngineWithTags(t *testing.T, eng *stats.Engine) {
	e2 := eng.WithTags(
		stats.T("command", "hello world"),
		stats.T("system", "testing"),
	)

	if e2.Prefix != "test" {
		t.Error("bad prefix:", e2.Prefix)
	}

	if !reflect.DeepEqual(e2.Tags, []stats.Tag{
		stats.T("command", "hello world"),
		stats.T("service", "test-service"),
		stats.T("system", "testing"),
	}) {
		t.Error("bad tags:", e2.Tags)
	}
}

func testEngineFlush(t *testing.T, eng *stats.Engine) {
	eng.Flush()
	eng.Flush()
	eng.Flush()

	h := eng.Handler.(*statstest.Handler)

	if n := h.FlushCalls(); n != 3 {
		t.Error("bad number of flush calls:", n)
	}
}

func testEngineIncr(t *testing.T, eng *stats.Engine) {
	eng.Incr("measure.count")
	eng.Incr("measure.count", stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service")},
		},
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineAdd(t *testing.T, eng *stats.Engine) {
	eng.Add("measure.count", 42)
	eng.Add("measure.count", 10, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("count", 42, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service")},
		},
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("count", 10, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineSet(t *testing.T, eng *stats.Engine) {
	eng.Set("measure.level", 42)
	eng.Set("measure.level", 10, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("level", 42, stats.Gauge)},
			Tags:   []stats.Tag{stats.T("service", "test-service")},
		},
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("level", 10, stats.Gauge)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineObserve(t *testing.T, eng *stats.Engine) {
	eng.Observe("measure.size", 42)
	eng.Observe("measure.size", 10, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("size", 42, stats.Histogram)},
			Tags:   []stats.Tag{stats.T("service", "test-service")},
		},
		stats.Measure{
			Name:   "test.measure",
			Fields: []stats.Field{stats.MakeField("size", 10, stats.Histogram)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineReport(t *testing.T, eng *stats.Engine) {
	m := struct {
		Count int `metric:"count" type:"counter"`
	}{42}

	eng.Report(m)
	eng.Report(m, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 42, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service")},
		},
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 42, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineReportArray(t *testing.T, eng *stats.Engine) {
	m := [2]struct {
		Count int `metric:"count" type:"counter"`
	}{}
	m[0].Count = 1
	m[1].Count = 2

	eng.Report(m, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 2, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineReportSlice(t *testing.T, eng *stats.Engine) {
	m := []struct {
		Count int `metric:"count" type:"counter"`
	}{{}, {}}
	m[0].Count = 1
	m[1].Count = 2

	eng.Report(m, stats.T("type", "testing"))

	checkMeasuresEqual(t, eng,
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
		stats.Measure{
			Name:   "test",
			Fields: []stats.Field{stats.MakeField("count", 2, stats.Counter)},
			Tags:   []stats.Tag{stats.T("service", "test-service"), stats.T("type", "testing")},
		},
	)
}

func testEngineClock(t *testing.T, eng *stats.Engine) {
	c := eng.Clock("upload", stats.T("f", "img.jpg"))
	c.Stamp("compress")
	c.Stamp("grayscale")
	c.Stop()

	found := measures(t, eng)

	if len(found) != 3 {
		t.Fatalf("expected 3 measures got %d", len(found))
	}

	stamps := []string{"compress", "grayscale", "total"}

	for i, m := range found {
		if m.Name != "test" {
			t.Errorf("measure name mismatch, got %q", m.Name)
		}

		if len(m.Tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(m.Tags))
		}

		exp := []stats.Tag{
			stats.T("f", "img.jpg"),
			stats.T("service", "test-service"),
			stats.T("stamp", stamps[i]),
		}

		if !reflect.DeepEqual(m.Tags, exp) {
			t.Errorf("tag mismatch, expected %v, got %v", exp, m.Tags)
		}
	}
}

func checkMeasuresEqual(t *testing.T, eng *stats.Engine, expected ...stats.Measure) {
	found := measures(t, eng)
	if !reflect.DeepEqual(found, expected) {
		t.Error("bad measures:")
		t.Logf("expected: %#v", expected)
		t.Logf("found:    %#v", found)
	}
}

func measures(t *testing.T, eng *stats.Engine) []stats.Measure {
	return eng.Handler.(*statstest.Handler).Measures()
}

func BenchmarkEngine(b *testing.B) {
	engines := []struct {
		name  string
		value stats.Engine
	}{
		{
			name:  "discard",
			value: stats.Engine{Handler: stats.Discard},
		},
		{
			name: "datadog",
			value: stats.Engine{Handler: datadog.NewClientWith(datadog.ClientConfig{
				BufferSize: datadog.MaxBufferSize,
			})},
		},
		{
			name: "influxdb",
			value: stats.Engine{Handler: influxdb.NewClientWith(influxdb.ClientConfig{
				Transport: &discardTransport{},
			})},
		},
		{
			name:  "prometheus",
			value: stats.Engine{Handler: &prometheus.Handler{}},
		},
	}

	for _, eng := range engines {
		b.Run(eng.name, func(b *testing.B) {
			tests := []struct {
				scenario string
				function func(*testing.PB, *stats.Engine)
			}{
				{
					scenario: "Engine.Add.1x",
					function: benchmarkEngineAdd1x,
				},
				{
					scenario: "Engine.Set.1x",
					function: benchmarkEngineSet1x,
				},
				{
					scenario: "Engine.Observe.1x",
					function: benchmarkEngineObserve1x,
				},
				{
					scenario: "Engine.Add.10x",
					function: benchmarkEngineAdd10x,
				},
				{
					scenario: "Engine.Set.10x",
					function: benchmarkEngineSet10x,
				},
				{
					scenario: "Engine.Observe.10x",
					function: benchmarkEngineObserve10x,
				},
				{
					scenario: "Engine.ReportAt(struct)",
					function: benchmarkEngineReportAtStruct,
				},
				{
					scenario: "Engine.ReportAt(struct:large)",
					function: benchmarkEngineReportAtStructLarge,
				},
				{
					scenario: "Engine.ReportAt(array)",
					function: benchmarkEngineReportAtArray,
				},
				{
					scenario: "Engine.ReportAt(slice)",
					function: benchmarkEngineReportAtSlice,
				},
			}

			for _, test := range tests {
				b.Run(test.scenario, func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) { test.function(pb, &eng.value) })
				})
			}
		})
	}
}

func benchmarkEngineAdd1x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Add("calls", 1)
	}
}

func benchmarkEngineSet1x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Set("calls", 1)
	}
}

func benchmarkEngineObserve1x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Observe("calls", 1)
	}
}

func benchmarkEngineAdd10x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
		e.Add("calls", 1)
	}
}

func benchmarkEngineSet10x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
		e.Set("calls", 1)
	}
}

func benchmarkEngineObserve10x(pb *testing.PB, e *stats.Engine) {
	for pb.Next() {
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
		e.Observe("calls", 1)
	}
}

func benchmarkEngineReportAtStruct(pb *testing.PB, e *stats.Engine) {
	t := time.Now()
	m := struct {
		Calls int `metric:"calls" type:"counter"`
	}{1}

	for pb.Next() {
		e.ReportAt(t, &m)
	}
}

func benchmarkEngineReportAtStructLarge(pb *testing.PB, e *stats.Engine) {
	t := time.Now()
	m := struct {
		Calls0 int `metric:"calls" type:"counter"`
		Calls1 int `metric:"calls" type:"counter"`
		Calls2 int `metric:"calls" type:"counter"`
		Calls3 int `metric:"calls" type:"counter"`
		Calls4 int `metric:"calls" type:"counter"`
		Calls5 int `metric:"calls" type:"counter"`
		Calls6 int `metric:"calls" type:"counter"`
		Calls7 int `metric:"calls" type:"counter"`
		Calls8 int `metric:"calls" type:"counter"`
		Calls9 int `metric:"calls" type:"counter"`
	}{}

	for pb.Next() {
		e.ReportAt(t, &m)
	}
}

func benchmarkEngineReportAtArray(pb *testing.PB, e *stats.Engine) {
	t := time.Now()
	m := [1]struct {
		Calls int `metric:"calls" type:"counter"`
	}{}
	m[0].Calls = 1

	for pb.Next() {
		e.ReportAt(t, &m)
	}
}

func benchmarkEngineReportAtSlice(pb *testing.PB, e *stats.Engine) {
	t := time.Now()
	m := []struct {
		Calls int `metric:"calls" type:"counter"`
	}{{}}
	m[0].Calls = 1

	for pb.Next() {
		e.ReportAt(t, &m)
	}
}

type discardTransport struct{}

func (t *discardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}
