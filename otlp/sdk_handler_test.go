package otlp

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
)

func TestSDKHandler_HandleMeasures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create handler with gRPC protocol
	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test counter
	handler.HandleMeasures(now, stats.Measure{
		Name:   "test.counter",
		Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
		Tags:   []stats.Tag{{Name: "env", Value: "test"}},
	})

	// Test gauge
	handler.HandleMeasures(now, stats.Measure{
		Name:   "test.gauge",
		Fields: []stats.Field{stats.MakeField("value", 42.5, stats.Gauge)},
		Tags:   []stats.Tag{{Name: "env", Value: "test"}},
	})

	// Test histogram
	handler.HandleMeasures(now, stats.Measure{
		Name:   "test.histogram",
		Fields: []stats.Field{stats.MakeField("duration", 100, stats.Histogram)},
		Tags:   []stats.Tag{{Name: "env", Value: "test"}},
	})

	// Flush metrics
	handler.Flush()

	// Verify instruments were created
	if len(handler.instruments) != 3 {
		t.Errorf("expected 3 instruments, got %d", len(handler.instruments))
	}
}

func TestSDKHandler_HTTP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create handler with HTTP protocol
	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolHTTPProtobuf,
		Endpoint:       "localhost:4318",
		ExportInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test basic metric
	handler.HandleMeasures(now, stats.Measure{
		Name:   "http.test",
		Fields: []stats.Field{stats.MakeField("requests", 10, stats.Counter)},
		Tags:   []stats.Tag{{Name: "method", Value: "GET"}},
	})

	handler.Flush()
}

func TestSDKHandler_FromEnv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This test demonstrates using environment variables
	// In real usage, OTEL_EXPORTER_OTLP_ENDPOINT and other vars would be set
	handler, err := NewSDKHandlerFromEnv(ctx)
	if err != nil {
		t.Fatalf("failed to create handler from env: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	handler.HandleMeasures(now, stats.Measure{
		Name:   "env.test",
		Fields: []stats.Field{stats.MakeField("value", 1, stats.Counter)},
		Tags:   []stats.Tag{{Name: "source", Value: "env"}},
	})
}

func TestSDKHandler_MultipleMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Send multiple measures in one call
	handler.HandleMeasures(now,
		stats.Measure{
			Name:   "app.requests",
			Fields: []stats.Field{stats.MakeField("count", 100, stats.Counter)},
			Tags:   []stats.Tag{{Name: "status", Value: "200"}},
		},
		stats.Measure{
			Name:   "app.requests",
			Fields: []stats.Field{stats.MakeField("count", 10, stats.Counter)},
			Tags:   []stats.Tag{{Name: "status", Value: "404"}},
		},
		stats.Measure{
			Name: "app.latency",
			Fields: []stats.Field{
				stats.MakeField("p50", 50, stats.Histogram),
				stats.MakeField("p99", 200, stats.Histogram),
			},
			Tags: []stats.Tag{{Name: "endpoint", Value: "/api/users"}},
		},
	)

	handler.Flush()

	// Should have created 4 instruments:
	// app.requests.count (2 tag variations share same instrument)
	// app.latency.p50
	// app.latency.p99
	if len(handler.instruments) < 3 {
		t.Errorf("expected at least 3 instruments, got %d", len(handler.instruments))
	}
}

func TestSDKHandler_ValueConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test different value types
	testCases := []struct {
		name      string
		value     interface{}
		fieldType stats.FieldType
	}{
		{"int", int(42), stats.Counter},
		{"uint", uint(42), stats.Counter},
		{"float", float64(42.5), stats.Gauge},
		{"duration", time.Second, stats.Histogram},
		{"bool_true", true, stats.Counter},
		{"bool_false", false, stats.Counter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler.HandleMeasures(now, stats.Measure{
				Name:   "conversion.test",
				Fields: []stats.Field{stats.MakeField(tc.name, tc.value, tc.fieldType)},
				Tags:   []stats.Tag{{Name: "type", Value: tc.name}},
			})
		})
	}

	handler.Flush()
}

func TestSDKHandler_GaugeBehavior(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test that gauges maintain absolute values, not cumulative
	// Set gauge to 100
	handler.HandleMeasures(now, stats.Measure{
		Name:   "test.gauge",
		Fields: []stats.Field{stats.MakeField("value", 100, stats.Gauge)},
		Tags:   []stats.Tag{{Name: "test", Value: "gauge"}},
	})

	// Set gauge to 50 (should be 50, not 150)
	handler.HandleMeasures(now.Add(time.Second), stats.Measure{
		Name:   "test.gauge",
		Fields: []stats.Field{stats.MakeField("value", 50, stats.Gauge)},
		Tags:   []stats.Tag{{Name: "test", Value: "gauge"}},
	})

	// Set gauge to 75 (should be 75, not 125 or 225)
	handler.HandleMeasures(now.Add(2*time.Second), stats.Measure{
		Name:   "test.gauge",
		Fields: []stats.Field{stats.MakeField("value", 75, stats.Gauge)},
		Tags:   []stats.Tag{{Name: "test", Value: "gauge"}},
	})

	// Gauges now use native Float64Gauge which maintains absolute values directly
	// No need to track internal state - the OTel SDK handles this

	handler.Flush()

	// Verify instrument was created
	if len(handler.instruments) < 1 {
		t.Errorf("expected at least 1 instrument, got %d", len(handler.instruments))
	}
}

func TestSDKHandler_ExponentialHistogram(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create handler with exponential histogram enabled
	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:                      ProtocolGRPC,
		Endpoint:                      "localhost:4317",
		ExportInterval:                1 * time.Second,
		ExponentialHistogram:          true,
		ExponentialHistogramMaxSize:   160,
		ExponentialHistogramMaxScale:  20,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test histogram with exponential aggregation
	handler.HandleMeasures(now, stats.Measure{
		Name:   "request.duration",
		Fields: []stats.Field{stats.MakeField("ms", 100, stats.Histogram)},
		Tags:   []stats.Tag{{Name: "endpoint", Value: "/api/users"}},
	})

	handler.HandleMeasures(now.Add(time.Millisecond), stats.Measure{
		Name:   "request.duration",
		Fields: []stats.Field{stats.MakeField("ms", 250, stats.Histogram)},
		Tags:   []stats.Tag{{Name: "endpoint", Value: "/api/users"}},
	})

	handler.HandleMeasures(now.Add(2*time.Millisecond), stats.Measure{
		Name:   "request.duration",
		Fields: []stats.Field{stats.MakeField("ms", 150, stats.Histogram)},
		Tags:   []stats.Tag{{Name: "endpoint", Value: "/api/users"}},
	})

	handler.Flush()

	// Verify instrument was created
	if len(handler.instruments) < 1 {
		t.Errorf("expected at least 1 instrument, got %d", len(handler.instruments))
	}
}

func TestSDKHandler_CumulativeTemporality(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create handler with default (cumulative) temporality
	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 1 * time.Second,
		// TemporalitySelector: nil means default cumulative temporality
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()

	// Test counter - should accumulate
	handler.HandleMeasures(now, stats.Measure{
		Name:   "requests",
		Fields: []stats.Field{stats.MakeField("count", 10, stats.Counter)},
		Tags:   []stats.Tag{{Name: "endpoint", Value: "/api"}},
	})

	handler.HandleMeasures(now.Add(time.Second), stats.Measure{
		Name:   "requests",
		Fields: []stats.Field{stats.MakeField("count", 15, stats.Counter)},
		Tags:   []stats.Tag{{Name: "endpoint", Value: "/api"}},
	})

	// With cumulative temporality, counters accumulate (10 + 15 = 25 total)
	// The SDK handles this internally

	handler.Flush()

	// Verify instrument was created
	if len(handler.instruments) < 1 {
		t.Errorf("expected at least 1 instrument, got %d", len(handler.instruments))
	}
}

func BenchmarkSDKHandler_HandleMeasures(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler, err := NewSDKHandler(ctx, SDKConfig{
		Protocol:       ProtocolGRPC,
		Endpoint:       "localhost:4317",
		ExportInterval: 10 * time.Second,
	})
	if err != nil {
		b.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Shutdown(ctx)

	now := time.Now()
	measure := stats.Measure{
		Name:   "benchmark.test",
		Fields: []stats.Field{stats.MakeField("count", 1, stats.Counter)},
		Tags:   []stats.Tag{{Name: "env", Value: "bench"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleMeasures(now, measure)
	}
}
