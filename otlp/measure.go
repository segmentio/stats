package otlp

import (
	"github.com/vertoforce/stats"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

func convertMetrics(metrics ...metric) []*metricpb.Metric {
	mm := []*metricpb.Metric{}

	for _, metric := range metrics {
		attributes := tagsToAttributes(metric.tags...)

		m := &metricpb.Metric{
			Name: metric.measureName + "." + metric.fieldName,
		}

		switch metric.fieldType {
		case stats.Counter:
			if m.Data == nil {
				m.Data = &metricpb.Metric_Sum{
					Sum: &metricpb.Sum{
						AggregationTemporality: metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						DataPoints:             []*metricpb.NumberDataPoint{},
					},
				}
			}

			sum := m.GetSum()
			sum.DataPoints = append(sum.DataPoints, &metricpb.NumberDataPoint{
				TimeUnixNano: uint64(metric.time.UnixNano()),
				Value:        &metricpb.NumberDataPoint_AsDouble{AsDouble: valueOf(metric.value)},
				Attributes:   attributes,
			})
		case stats.Gauge:
			if m.Data == nil {
				m.Data = &metricpb.Metric_Gauge{
					Gauge: &metricpb.Gauge{
						DataPoints: []*metricpb.NumberDataPoint{},
					},
				}
			}

			gauge := m.GetGauge()
			gauge.DataPoints = append(gauge.DataPoints, &metricpb.NumberDataPoint{
				TimeUnixNano: uint64(metric.time.UnixNano()),
				Value:        &metricpb.NumberDataPoint_AsDouble{AsDouble: valueOf(metric.value)},
				Attributes:   attributes,
			})
		case stats.Histogram:
			if m.Data == nil {
				m.Data = &metricpb.Metric_Histogram{
					Histogram: &metricpb.Histogram{
						AggregationTemporality: metricpb.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
						DataPoints:             []*metricpb.HistogramDataPoint{},
					},
				}
			}

			explicitBounds := make([]float64, len(metric.buckets))
			bucketCounts := make([]uint64, len(metric.buckets))

			for i, b := range metric.buckets {
				explicitBounds[i] = b.upperBound
				bucketCounts[i] = b.count
			}

			histogram := m.GetHistogram()
			histogram.DataPoints = append(histogram.DataPoints, &metricpb.HistogramDataPoint{
				TimeUnixNano:   uint64(metric.time.UnixNano()),
				Sum:            &metric.sum,
				Count:          metric.count,
				ExplicitBounds: explicitBounds,
				BucketCounts:   bucketCounts,
			})

		default:
		}

		mm = append(mm, m)
	}

	return mm
}

func valueOf(v stats.Value) float64 {
	switch v.Type() {
	case stats.Bool:
		if v.Bool() {
			return 1.0
		}
	case stats.Int:
		return float64(v.Int())
	case stats.Uint:
		return float64(v.Uint())
	case stats.Float:
		return v.Float()
	case stats.Duration:
		return v.Duration().Seconds()
	}
	return 0.0
}

func tagsToAttributes(tags ...stats.Tag) []*commonpb.KeyValue {
	attr := make([]*commonpb.KeyValue, 0, len(tags))

	for _, tag := range tags {
		attr = append(attr, &commonpb.KeyValue{
			Key: tag.Name,
			Value: &commonpb.AnyValue{
				Value: &commonpb.AnyValue_StringValue{
					StringValue: tag.Value,
				},
			},
		})
	}

	return attr
}
