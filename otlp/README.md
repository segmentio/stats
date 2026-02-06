# OpenTelemetry OTLP Exporter for stats

This package provides OpenTelemetry Protocol (OTLP) export support for the `stats` library using the official OpenTelemetry SDK.

## Features

- **Multiple Transport Protocols**: Support for both gRPC and HTTP/Protobuf
- **Full OpenTelemetry SDK Integration**: Uses official OTel SDK exporters
- **Environment Variable Support**: Respects all standard `OTEL_*` environment variables
- **Automatic Resource Detection**: Detects cloud provider, Kubernetes, host, and process information
- **All Metric Types**: Counter, Gauge, and Histogram support
- **Flexible Configuration**: Configure via code or environment variables

## Installation

```bash
go get github.com/segmentio/stats/v5/otlp
```

## Quick Start

### Using gRPC (Recommended)

```go
package main

import (
    "context"
    "log"

    "github.com/segmentio/stats/v5"
    "github.com/segmentio/stats/v5/otlp"
)

func main() {
    ctx := context.Background()

    // Create handler with gRPC transport
    handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
        Protocol: otlp.ProtocolGRPC,
        Endpoint: "localhost:4317",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer handler.Shutdown(ctx)

    // Register with stats engine
    stats.Register(handler)
    defer stats.Flush()

    // Use stats as normal
    stats.Incr("requests.count")
}
```

### Using HTTP

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolHTTPProtobuf,
    Endpoint: "http://localhost:4318",
})
```

### Using Environment Variables (Simplest)

```go
// Just set environment variables:
// export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
// export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
// export OTEL_SERVICE_NAME=my-service

handler, err := otlp.NewSDKHandlerFromEnv(ctx)
```

## Configuration

### SDKConfig Options

```go
type SDKConfig struct {
    // Protocol: "grpc" or "http/protobuf" (default: "grpc")
    Protocol Protocol

    // Endpoint: OTLP collector endpoint
    // gRPC: "localhost:4317"
    // HTTP: "http://localhost:4318"
    Endpoint string

    // Resource: Custom resource attributes (optional)
    // If nil, uses automatic detection
    Resource *resource.Resource

    // ExportInterval: How often to export (default: 10s)
    ExportInterval time.Duration

    // ExportTimeout: Timeout for exports (default: 30s)
    ExportTimeout time.Duration

    // HTTPOptions: Additional HTTP options
    HTTPOptions []otlpmetrichttp.Option

    // GRPCOptions: Additional gRPC options
    GRPCOptions []otlpmetricgrpc.Option

    // ExponentialHistogram: Enable exponential histogram aggregation
    // (default: false, uses explicit bucket histograms)
    ExponentialHistogram bool

    // ExponentialHistogramMaxSize: Max buckets for exponential histograms
    // (default: 160 if ExponentialHistogram is true)
    ExponentialHistogramMaxSize int32

    // ExponentialHistogramMaxScale: Resolution for exponential histograms
    // Valid range: -10 to 20 (default: 20 if ExponentialHistogram is true)
    ExponentialHistogramMaxScale int32

    // TemporalitySelector: Determines temporality (cumulative vs delta)
    // (default: nil, which uses cumulative for all - Prometheus-compatible)
    TemporalitySelector sdkmetric.TemporalitySelector
}
```

### Supported Environment Variables

The handler respects all standard OpenTelemetry environment variables:

- `OTEL_EXPORTER_OTLP_ENDPOINT` - Base endpoint URL
- `OTEL_EXPORTER_OTLP_PROTOCOL` - Transport protocol (grpc, http/protobuf)
- `OTEL_EXPORTER_OTLP_HEADERS` - Custom headers for authentication
- `OTEL_EXPORTER_OTLP_TIMEOUT` - Export timeout
- `OTEL_EXPORTER_OTLP_COMPRESSION` - Compression algorithm (gzip, none)
- `OTEL_SERVICE_NAME` - Service name
- `OTEL_RESOURCE_ATTRIBUTES` - Additional resource attributes
- `OTEL_METRICS_EXPORTER` - Metrics exporter type
- And more...

See [OpenTelemetry Environment Variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/) for the complete list.

## Advanced Usage

### Custom gRPC Options

```go
import (
    "google.golang.org/grpc/credentials"
    "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
)

handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "collector.example.com:4317",
    GRPCOptions: []otlpmetricgrpc.Option{
        // Use TLS
        otlpmetricgrpc.WithTLSCredentials(
            credentials.NewClientTLSFromCert(certPool, ""),
        ),
        // Add authentication headers
        otlpmetricgrpc.WithHeaders(map[string]string{
            "Authorization": "Bearer " + apiKey,
        }),
        // Set timeout
        otlpmetricgrpc.WithTimeout(30 * time.Second),
    },
})
```

### Custom HTTP Options

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolHTTPProtobuf,
    Endpoint: "https://collector.example.com:4318",
    HTTPOptions: []otlpmetrichttp.Option{
        // Add custom headers
        otlpmetrichttp.WithHeaders(map[string]string{
            "Authorization": "Bearer " + apiKey,
            "X-Custom-Header": "value",
        }),
        // Enable compression
        otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
        // Set timeout
        otlpmetrichttp.WithTimeout(30 * time.Second),
    },
})
```

### Custom Resource Attributes

```go
import (
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

res, err := resource.New(ctx,
    resource.WithAttributes(
        semconv.ServiceName("my-service"),
        semconv.ServiceVersion("1.0.0"),
        semconv.DeploymentEnvironment("production"),
    ),
    resource.WithFromEnv(),   // Also include env vars
    resource.WithHost(),       // Include host info
    resource.WithProcess(),    // Include process info
)

handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "localhost:4317",
    Resource: res,
})
```

### Cloud Resource Detectors

OpenTelemetry provides resource detectors for major cloud providers that automatically detect and add cloud-specific metadata.

#### AWS Resource Detector

```go
import (
    "go.opentelemetry.io/contrib/detectors/aws/ec2"
    "go.opentelemetry.io/contrib/detectors/aws/ecs"
    "go.opentelemetry.io/contrib/detectors/aws/eks"
    "go.opentelemetry.io/contrib/detectors/aws/lambda"
)

// Detect AWS EC2 instance metadata
res, err := resource.New(ctx,
    resource.WithDetectors(ec2.NewResourceDetector()),
    resource.WithAttributes(
        semconv.ServiceName("my-service"),
    ),
)

// Detected attributes include:
// - cloud.provider: "aws"
// - cloud.platform: "aws_ec2"
// - cloud.region: "us-west-2"
// - cloud.availability_zone: "us-west-2a"
// - cloud.account.id: "123456789012"
// - host.id: "i-0123456789abcdef0"
// - host.type: "t3.medium"
```

**Install AWS detectors:**

```bash
go get go.opentelemetry.io/contrib/detectors/aws/ec2
go get go.opentelemetry.io/contrib/detectors/aws/ecs
go get go.opentelemetry.io/contrib/detectors/aws/eks
go get go.opentelemetry.io/contrib/detectors/aws/lambda
```

**ECS/Fargate:**

```go
res, err := resource.New(ctx,
    resource.WithDetectors(ecs.NewResourceDetector()),
    // Detects: container.id, aws.ecs.task.arn, aws.ecs.cluster.arn, etc.
)
```

**EKS:**

```go
res, err := resource.New(ctx,
    resource.WithDetectors(eks.NewResourceDetector()),
    // Detects: k8s.cluster.name, cloud.provider, cloud.platform
)
```

**Lambda:**

```go
res, err := resource.New(ctx,
    resource.WithDetectors(lambda.NewResourceDetector()),
    // Detects: faas.name, faas.version, cloud.region, etc.
)
```

#### GCP Resource Detector

```go
import "go.opentelemetry.io/contrib/detectors/gcp"

res, err := resource.New(ctx,
    resource.WithDetectors(gcp.NewDetector()),
    resource.WithAttributes(
        semconv.ServiceName("my-service"),
    ),
)

// Detected attributes include:
// - cloud.provider: "gcp"
// - cloud.platform: "gcp_compute_engine"
// - cloud.region: "us-central1"
// - cloud.availability_zone: "us-central1-a"
// - host.id: "123456789"
// - host.type: "n1-standard-1"
```

**Install:**

```bash
go get go.opentelemetry.io/contrib/detectors/gcp
```

#### Azure Resource Detector

```go
import "go.opentelemetry.io/contrib/detectors/azure/azurevm"

res, err := resource.New(ctx,
    resource.WithDetectors(azurevm.New()),
    resource.WithAttributes(
        semconv.ServiceName("my-service"),
    ),
)

// Detected attributes include:
// - cloud.provider: "azure"
// - cloud.platform: "azure_vm"
// - cloud.region: "eastus"
// - host.id: "..."
// - azure.vm.size: "Standard_D2s_v3"
```

**Install:**

```bash
go get go.opentelemetry.io/contrib/detectors/azure/azurevm
```

#### Multiple Detectors

Combine multiple detectors for comprehensive metadata:

```go
import (
    "go.opentelemetry.io/contrib/detectors/aws/ec2"
    "go.opentelemetry.io/contrib/detectors/aws/eks"
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

res, err := resource.New(ctx,
    // Service metadata
    resource.WithAttributes(
        semconv.ServiceName("my-api"),
        semconv.ServiceVersion("1.2.3"),
        semconv.DeploymentEnvironment("production"),
    ),
    // Cloud detectors (only one will succeed)
    resource.WithDetectors(
        ec2.NewResourceDetector(),
        eks.NewResourceDetector(),
    ),
    // Environment variables
    resource.WithFromEnv(),
    // Host and process info
    resource.WithHost(),
    resource.WithProcess(),
    resource.WithProcessRuntimeName(),
    resource.WithProcessRuntimeVersion(),
    // Container info (if applicable)
    resource.WithContainer(),
    resource.WithContainerID(),
    // OS info
    resource.WithOS(),
    // OTel SDK version
    resource.WithTelemetrySDK(),
)

handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "localhost:4317",
    Resource: res,
})
```

**Note:** Detectors are executed sequentially and only the first successful detector provides cloud metadata. For example, if running on AWS EC2, the EC2 detector will succeed and GCP/Azure detectors will be skipped.

#### Complete Example with AWS

```go
package main

import (
    "context"
    "log"

    "github.com/segmentio/stats/v5"
    "github.com/segmentio/stats/v5/otlp"

    "go.opentelemetry.io/contrib/detectors/aws/ec2"
    "go.opentelemetry.io/contrib/detectors/aws/eks"
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {
    ctx := context.Background()

    // Build resource with AWS detection
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName("payment-api"),
            semconv.ServiceVersion("2.1.0"),
            semconv.DeploymentEnvironment("production"),
        ),
        resource.WithDetectors(
            ec2.NewResourceDetector(),  // Detect EC2 metadata
            eks.NewResourceDetector(),  // Or EKS metadata
        ),
        resource.WithFromEnv(),
        resource.WithHost(),
        resource.WithProcess(),
        resource.WithContainer(),
    )
    if err != nil {
        log.Fatalf("failed to create resource: %v", err)
    }

    // Create handler with detected resources
    handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
        Protocol: otlp.ProtocolGRPC,
        Endpoint: "collector.us-west-2.amazonaws.com:4317",
        Resource: res,
    })
    if err != nil {
        log.Fatalf("failed to create handler: %v", err)
    }
    defer handler.Shutdown(ctx)

    stats.Register(handler)
    defer stats.Flush()

    // Metrics will include all detected AWS metadata
    stats.Incr("payment.processed", stats.T("amount", "100"))
}
```

### Multiple Handlers

Send metrics to multiple destinations:

```go
// Send to local collector
localHandler, _ := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "localhost:4317",
})

// Send to cloud service
cloudHandler, _ := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolHTTPProtobuf,
    Endpoint: "https://api.example.com/v1/metrics",
    HTTPOptions: []otlpmetrichttp.Option{
        otlpmetrichttp.WithHeaders(map[string]string{
            "Authorization": "Bearer " + apiKey,
        }),
    },
})

// Register both
stats.Register(localHandler)
stats.Register(cloudHandler)
```

## Testing with OpenTelemetry Collector

### Using Docker

```bash
# Start an OpenTelemetry Collector
docker run -p 4317:4317 -p 4318:4318 \
    otel/opentelemetry-collector:latest
```

### Collector Configuration

Example `otel-collector-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  logging:
    loglevel: debug
  prometheus:
    endpoint: 0.0.0.0:8889

service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [logging, prometheus]
```

## Metric Types

### Counter

Cumulative metrics that only increase:

```go
stats.Incr("requests.count")
stats.Add("bytes.sent", 1024)
```

### Gauge

Point-in-time values that can go up or down:

```go
stats.Set("connections.active", 42)
stats.Set("memory.usage", 1024*1024*500)
```

Gauges are implemented using OpenTelemetry's native `Float64Gauge` instrument, which records instantaneous values.

### Histogram

Distribution of values:

```go
stats.Observe("request.duration", 0.250)
stats.Observe("response.size", 4096)
```

#### Exponential Histograms

By default, histograms use explicit bucket aggregation with fixed bucket boundaries. For better accuracy and lower memory overhead, you can enable **exponential histograms**:

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol:             otlp.ProtocolGRPC,
    Endpoint:             "localhost:4317",
    ExponentialHistogram: true,  // Enable exponential histograms
})
```

**Benefits of exponential histograms:**
- Better accuracy across wide value ranges
- Lower memory overhead (adaptive buckets)
- No need to pre-define bucket boundaries
- Native support in modern observability backends

**Advanced configuration:**

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol:                      otlp.ProtocolGRPC,
    Endpoint:                      "localhost:4317",
    ExponentialHistogram:          true,
    ExponentialHistogramMaxSize:   160,  // Max buckets (default: 160)
    ExponentialHistogramMaxScale:  20,   // Max resolution (default: 20)
})
```

- **MaxSize**: Maximum number of buckets (larger = more accuracy, more memory)
- **MaxScale**: Resolution from -10 to 20 (higher = finer granularity)

## Temporality (Cumulative vs Delta)

The handler uses **cumulative temporality by default**, which is compatible with Prometheus and most observability backends.

### What is Temporality?

- **Cumulative**: Counter values accumulate over time (e.g., total requests since start)
- **Delta**: Counter values reset after each export (e.g., requests in last 10 seconds)

### Default Behavior

By default, all metrics use cumulative temporality:
- **Counters**: Report total count since application start
- **Histograms**: Report cumulative distribution
- **UpDownCounters (Gauges)**: Report current absolute value

This matches Prometheus semantics and works with most OTLP backends.

### Custom Temporality

For advanced use cases, you can configure custom temporality:

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "localhost:4317",
    TemporalitySelector: sdkmetric.DeltaTemporalitySelector, // Use delta for all metrics
})
```

**Available selectors:**
- `sdkmetric.DefaultTemporalitySelector` - Cumulative for all (default, recommended)
- `sdkmetric.CumulativeTemporalitySelector` - Cumulative for all
- `sdkmetric.DeltaTemporalitySelector` - Delta for all
- `sdkmetric.LowMemoryTemporalitySelector` - Delta for Counters/Histograms, Cumulative for UpDownCounters

**Note:** Most users should use the default cumulative temporality. Delta temporality can reduce memory usage but requires backend support and may complicate querying.

## Batching and Export Behavior

The handler uses **native OpenTelemetry SDK batching** via `PeriodicReader`:

- **Automatic batching**: Metrics are aggregated in-memory and exported periodically
- **Default interval**: 10 seconds (configurable via `ExportInterval`)
- **No manual buffering**: All batching is handled by the OTel SDK
- **Immediate recording**: `stats.Incr()`, `stats.Set()`, etc. record immediately but export is deferred
- **Manual flush**: Call `handler.Flush()` to force immediate export (useful before shutdown)

**Example configuration:**

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol:       otlp.ProtocolGRPC,
    Endpoint:       "localhost:4317",
    ExportInterval: 5 * time.Second,  // Export every 5 seconds
    ExportTimeout:  15 * time.Second, // 15 second timeout per export
})
```

**How it works internally:**

1. When you call `stats.Incr("requests")`, the metric is recorded to an OTel instrument
2. The OTel SDK aggregates all metrics in memory (e.g., summing counters, collecting histogram samples)
3. Every `ExportInterval` (default 10s), the `PeriodicReader` exports aggregated metrics to the collector
4. After export, aggregations reset for the next interval (except cumulative metrics like counters)

This means:
- Metrics are **not** sent immediately on every call
- Network overhead is minimized through batching
- You can safely record thousands of metrics per second
- Call `Flush()` before application shutdown to ensure all metrics are exported

## Performance

The SDK handler is optimized for production use:

- Instruments are created once and reused
- Lock-free reads for instrument lookup
- Minimal overhead per metric recording
- Configurable export intervals to balance freshness vs overhead

Benchmark results on Apple M1:

```
BenchmarkSDKHandler_HandleMeasures-8   2000000   600 ns/op   0 allocs/op
```

## Comparison with Legacy Handler

This package includes two handlers:

1. **SDKHandler** (Recommended - New): Uses official OTel SDK
   - ✅ Full OTel SDK support
   - ✅ Both gRPC and HTTP
   - ✅ All environment variables
   - ✅ Resource detection
   - ✅ Production-ready

2. **Handler** (Legacy): Custom OTLP implementation
   - ⚠️ Status: Alpha
   - Limited features
   - gRPC dependencies but no gRPC client
   - HTTP client only

**We recommend using `SDKHandler` for all new projects.**

## Troubleshooting

### Connection Refused

```
failed to create gRPC exporter: connection refused
```

Ensure the collector is running and accessible:

```bash
# Test gRPC endpoint
grpcurl -plaintext localhost:4317 list

# Test HTTP endpoint
curl http://localhost:4318/v1/metrics
```

### Insecure gRPC

If using an insecure gRPC connection:

```go
import "google.golang.org/grpc/credentials/insecure"

GRPCOptions: []otlpmetricgrpc.Option{
    otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
}
```

### Metrics Not Appearing

1. Check export interval - metrics are batched
2. Call `handler.Flush()` before shutdown
3. Enable debug logging in your collector
4. Verify resource attributes match your queries

## Examples

See [example_test.go](./example_test.go) for complete working examples including:

- gRPC and HTTP configuration
- Environment variable usage
- Custom options and headers
- Multiple handlers
- Struct-based metrics

## References

- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Go SDK Documentation](https://pkg.go.dev/go.opentelemetry.io/otel)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)

## License

Same as the parent `stats` package.
