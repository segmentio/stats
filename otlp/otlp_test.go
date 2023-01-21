package otlp

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/vertoforce/stats"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

type testCase struct {
	in  []stats.Measure
	out []*metricpb.Metric
}

var (
	now         = time.Now()
	handleTests = []testCase{
		{
			in: []stats.Measure{
				{
					Name:   "foobar",
					Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
					Tags:   []stats.Tag{{Name: "env", Value: "dev"}},
				},
				{
					Name:   "foobar",
					Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
					Tags:   []stats.Tag{{Name: "env", Value: "dev"}},
				},
			},
			out: []*metricpb.Metric{
				{
					Name: "foobar.count",
					Data: &metricpb.Metric_Sum{
						Sum: &metricpb.Sum{
							AggregationTemporality: metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
							DataPoints: []*metricpb.NumberDataPoint{
								{
									TimeUnixNano: uint64(now.UnixNano()),
									Value:        &metricpb.NumberDataPoint_AsDouble{AsDouble: 2},
									Attributes:   tagsToAttributes(stats.T("env", "dev")),
								},
							},
						},
					},
				},
			},
		},
		{
			in: []stats.Measure{
				{
					Name: "foobar",
					Fields: []stats.Field{
						stats.MakeField("hist", 5, stats.Histogram),
						stats.MakeField("hist", 10, stats.Histogram),
						stats.MakeField("hist", 20, stats.Histogram),
					},
					Tags: []stats.Tag{{Name: "region", Value: "us-west-2"}},
				},
			},
			out: []*metricpb.Metric{
				{
					Name: "foobar.hist",
					Data: &metricpb.Metric_Histogram{
						Histogram: &metricpb.Histogram{
							AggregationTemporality: metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
							DataPoints: []*metricpb.HistogramDataPoint{
								{
									TimeUnixNano:   uint64(now.UnixNano()),
									Count:          3,
									Sum:            sumPtr(35),
									BucketCounts:   []uint64{0, 2, 1, 0},
									ExplicitBounds: []float64{0, 10, 100, 1000},
								},
							},
						},
					},
				},
			},
		},
		{
			in: []stats.Measure{
				{
					Name: "foobar",
					Fields: []stats.Field{
						stats.MakeField("gauge", 42, stats.Gauge),
					},
					Tags: []stats.Tag{{Name: "env", Value: "dev"}},
				},
			},
			out: []*metricpb.Metric{
				{
					Name: "foobar.gauge",
					Data: &metricpb.Metric_Gauge{
						Gauge: &metricpb.Gauge{
							DataPoints: []*metricpb.NumberDataPoint{
								{
									TimeUnixNano: uint64(now.UnixNano()),
									Value:        &metricpb.NumberDataPoint_AsDouble{AsDouble: 42},
									Attributes:   tagsToAttributes(stats.T("env", "dev")),
								},
							},
						},
					},
				},
			},
		},
	}
)

func sumPtr(f float64) *float64 {
	return &f
}

var conversionTests = []testCase{}

func initTest() {
	stats.Buckets.Set("foobar.hist",
		0,
		10,
		100,
		1000,
	)
}

func TestHandler(t *testing.T) {
	initTest()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	for i, test := range handleTests {
		t.Run(fmt.Sprintf("handle-%d", i), func(t *testing.T) {
			h := Handler{
				Client: &client{
					expected: test.out,
				},
				Context: ctx,
			}

			h.handleMeasures(now, test.in...)

			if err := h.flush(); err != nil {
				t.Error(err)
			}
		})
	}
}

type client struct {
	expected []*metricpb.Metric
}

func (c *client) Handle(ctx context.Context, request *colmetricpb.ExportMetricsServiceRequest) error {
	for _, rm := range request.GetResourceMetrics() {
		for _, sm := range rm.GetScopeMetrics() {
			metrics := sm.GetMetrics()
			if !reflect.DeepEqual(metrics, c.expected) {
				return fmt.Errorf(
					"unexpected metrics in request\nexpected: %v\ngot:%v\n",
					c.expected,
					metrics,
				)
			}
		}
	}
	return nil
}

// run go test -with-collector with a running local otel collector to help with testing.
var withCollector = flag.Bool("with-collector", false, "send metrics to a local collector")

func TestSendOtel(t *testing.T) {
	if !*withCollector {
		t.SkipNow()
	}

	initTest()
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	h := Handler{
		Client: &HTTPClient{
			client:   http.DefaultClient,
			endpoint: "http://localhost:4318/v1/metrics",
		},
		Context:    ctx,
		MaxMetrics: 10,
	}

	for i, test := range handleTests {
		t.Run(fmt.Sprintf("handle-%d", i), func(t *testing.T) {
			h.HandlerMeasure(now, test.in...)
		})
	}

	if err := h.flush(); err != nil {
		t.Error(err)
	}
}
