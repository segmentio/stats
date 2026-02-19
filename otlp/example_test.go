package otlp_test

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/stats/v5"
	"github.com/segmentio/stats/v5/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"google.golang.org/grpc/credentials/insecure"
)

// Example_gRPC demonstrates using the OpenTelemetry SDK handler with gRPC transport.
func Example_gRPC() {
	ctx := context.Background()

	// Create handler with gRPC transport
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolGRPC,
		EndpointURL: "http://localhost:4317",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	// Register with the default stats engine
	stats.Register(handler)
	defer stats.Flush()

	// Your application metrics will now be exported via gRPC
	stats.Incr("requests.count", stats.T("method", "GET"), stats.T("status", "200"))
	stats.Observe("request.duration", 0.250, stats.T("endpoint", "/api/users"))
}

// Example_hTTP demonstrates using the OpenTelemetry SDK handler with HTTP transport.
func Example_hTTP() {
	ctx := context.Background()

	// Create handler with HTTP transport
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolHTTPProtobuf,
		EndpointURL: "http://localhost:4318",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	// Register with the default stats engine
	stats.Register(handler)
	defer stats.Flush()

	// Your application metrics will now be exported via HTTP
	stats.Incr("requests.count")
}

// Example_fromEnv demonstrates using environment variables for configuration.
// This is the simplest approach and follows OpenTelemetry best practices.
func Example_fromEnv() {
	// Set environment variables before running:
	// export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
	// export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
	// export OTEL_SERVICE_NAME=my-service
	// export OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production,service.version=1.0.0

	ctx := context.Background()

	// Handler automatically reads all OTEL_* environment variables
	handler, err := otlp.NewSDKHandlerFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	stats.Incr("app.started")
}

// Example_fullyConfiguredByEnvironment demonstrates relying entirely on OTEL environment variables
// without specifying any configuration in code. This provides maximum flexibility for deployment
// environments to control OpenTelemetry configuration without code changes.
func Example_fullyConfiguredByEnvironment() {
	// The SDK will use these standard OpenTelemetry environment variables:
	//
	// Required/Common:
	//   OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 (full URL with scheme)
	//   OTEL_EXPORTER_OTLP_PROTOCOL=grpc (or http/protobuf)
	//   OTEL_SERVICE_NAME=my-service
	//
	// Optional:
	//   OTEL_EXPORTER_OTLP_HEADERS=key1=value1,key2=value2
	//   OTEL_EXPORTER_OTLP_TIMEOUT=30s
	//   OTEL_RESOURCE_ATTRIBUTES=deployment.environment=production
	//   OTEL_METRIC_EXPORT_INTERVAL=60s
	//   OTEL_METRIC_EXPORT_TIMEOUT=30s
	//
	// If no environment variables are set, the SDK uses these defaults:
	//   - Endpoint: http://localhost:4317 (gRPC) or http://localhost:4318 (HTTP)
	//   - Protocol: grpc
	//   - Export Interval: 60 seconds
	//   - Export Timeout: 30 seconds

	ctx := context.Background()

	// Pass an empty config - SDK will read all configuration from environment
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	// Your application code remains environment-agnostic
	stats.Incr("requests.count", stats.T("method", "GET"))
	stats.Observe("request.duration", 0.125)
}

// Example_gRPCWithOptions demonstrates advanced gRPC configuration.
func Example_gRPCWithOptions() {
	ctx := context.Background()

	// Create handler with custom gRPC options
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolGRPC,
		EndpointURL: "http://localhost:4317",
		GRPCOptions: []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithTimeout(30 * time.Second),
			// For TLS:
			// otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(certPool, "")),
			// For custom headers:
			// otlpmetricgrpc.WithHeaders(map[string]string{
			//     "Authorization": "Bearer token",
			// }),
		},
		ExportInterval: 10 * time.Second,
		ExportTimeout:  30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	stats.Incr("requests.total")
}

// Example_hTTPWithOptions demonstrates advanced HTTP configuration.
func Example_hTTPWithOptions() {
	ctx := context.Background()

	// Create handler with custom HTTP options
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolHTTPProtobuf,
		EndpointURL: "http://localhost:4318",
		HTTPOptions: []otlpmetrichttp.Option{
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithTimeout(30 * time.Second),
			// For custom headers:
			// otlpmetrichttp.WithHeaders(map[string]string{
			//     "Authorization": "Bearer token",
			//     "X-Custom-Header": "value",
			// }),
			// For compression:
			// otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		},
		ExportInterval: 10 * time.Second,
		ExportTimeout:  30 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	stats.Incr("requests.total")
}

// Example_multipleHandlers demonstrates using multiple handlers simultaneously.
func Example_multipleHandlers() {
	ctx := context.Background()

	// Send metrics to both gRPC and HTTP endpoints
	grpcHandler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolGRPC,
		EndpointURL: "http://localhost:4317",
		GRPCOptions: []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer grpcHandler.Shutdown(ctx)

	httpHandler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolHTTPProtobuf,
		EndpointURL: "http://localhost:4318",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer httpHandler.Shutdown(ctx)

	// Register both handlers
	stats.Register(grpcHandler)
	stats.Register(httpHandler)
	defer stats.Flush()

	// Metrics will be sent to both endpoints
	stats.Incr("requests.count")
}

// Example_structBased demonstrates using struct-based metrics with OpenTelemetry.
func Example_structBased() {
	ctx := context.Background()

	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol: otlp.ProtocolGRPC,
		EndpointURL: "http://localhost:4317",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	// Define metrics using struct tags
	type ServerMetrics struct {
		RequestCount    int           `metric:"requests.count" type:"counter"`
		ActiveConns     int           `metric:"connections.active" type:"gauge"`
		RequestDuration time.Duration `metric:"request.duration" type:"histogram"`
	}

	metrics := ServerMetrics{
		RequestCount:    100,
		ActiveConns:     50,
		RequestDuration: 250 * time.Millisecond,
	}

	// Report all metrics from the struct
	stats.Report(metrics, stats.T("server", "web-1"), stats.T("region", "us-west-2"))
}

func ExampleSDKHandler_exponentialHistogram() {
	ctx := context.Background()

	// Create handler with exponential histogram support
	// Exponential histograms provide better accuracy and lower memory overhead
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol:             otlp.ProtocolGRPC,
		EndpointURL:          "http://localhost:4317",
		ExponentialHistogram: true, // Enable exponential histograms
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	// Record histogram metrics - these will use exponential bucket aggregation
	stats.Observe("api.latency", 0.125, stats.T("endpoint", "/users"))
	stats.Observe("api.latency", 0.250, stats.T("endpoint", "/users"))
	stats.Observe("api.latency", 0.500, stats.T("endpoint", "/users"))

	// Exponential histograms automatically adapt to the value range
	// providing consistent accuracy without pre-defined bucket boundaries
	stats.Observe("db.query.duration", 0.001, stats.T("query", "SELECT"))
	stats.Observe("db.query.duration", 0.050, stats.T("query", "SELECT"))
	stats.Observe("db.query.duration", 1.500, stats.T("query", "SELECT"))
}

func ExampleSDKHandler_exponentialHistogramAdvanced() {
	ctx := context.Background()

	// Advanced exponential histogram configuration
	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
		Protocol:                      otlp.ProtocolGRPC,
		EndpointURL:                   "http://localhost:4317",
		ExponentialHistogram:          true,
		ExponentialHistogramMaxSize:   160, // Max buckets (higher = more accuracy)
		ExponentialHistogramMaxScale:  20,  // Max resolution (higher = finer granularity)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Shutdown(ctx)

	stats.Register(handler)
	defer stats.Flush()

	// Record response time metrics across wide value ranges
	// Exponential histograms handle this efficiently
	for _, duration := range []float64{0.001, 0.010, 0.100, 1.000, 10.000} {
		stats.Observe("response.time", duration, stats.T("service", "api"))
	}
}
