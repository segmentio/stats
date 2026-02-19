# OpenTelemetry SDK Implementation Notes

This document describes the implementation details and design decisions for the OpenTelemetry OTLP exporter.

## Overview

This implementation provides full OpenTelemetry Protocol (OTLP) support using the official OpenTelemetry SDK. It bridges the `stats` library's metric interface to OpenTelemetry's metric API.

## Architecture

### Core Components

1. **SDKHandler** - Main handler implementing `stats.Handler`
2. **Protocol Support** - Both gRPC and HTTP/Protobuf transports
3. **Instrument Management** - Efficient caching of OpenTelemetry instruments
4. **Gauge Value Tracking** - Delta calculation for absolute gauge semantics

## Design Decisions

### 1. Gauge Implementation

**Solution**: Use OpenTelemetry's native `Float64Gauge` instrument for synchronous gauge recording.

```go
// When stats.Set("metric", 42) is called:
gauge.Record(ctx, 42.0, opts)
```

**Why**: The OpenTelemetry SDK now provides native Gauge instruments that directly record instantaneous values. This provides the exact semantics users expect - `stats.Set("metric", 42)` records the value 42.

**Benefits**:
- No additional memory overhead for tracking previous values
- Direct mapping to OpenTelemetry's gauge semantics
- Simpler, more maintainable implementation

### 2. Context Management

**Challenge**: Stored contexts can be cancelled, causing metric recording to fail.

**Solution**:
- Use `context.Background()` for metric recording operations
- Store the initialization context as `shutdownCtx` only for shutdown operations
- This ensures metrics continue to be recorded even if the original context is cancelled

**Why**: Metric recording should be resilient and not fail due to context cancellation. The handler should continue working throughout the application lifecycle.

### 3. Instrument Caching

**Note**: This is an internal implementation detail - users don't need to worry about this.

**Implementation**: Thread-safe two-level locking pattern for efficient instrument reuse
```go
// Fast path: read lock for lookup (common case - instrument already exists)
h.mu.RLock()
inst, exists := h.instruments[metricName]
h.mu.RUnlock()

// Slow path: write lock only if creating new instrument (rare - first time seeing this metric)
if !exists {
    h.mu.Lock()
    // Double-check after acquiring write lock (another goroutine may have created it)
    inst, exists = h.instruments[metricName]
    if !exists {
        inst = h.createInstruments(meter, metricName, field.Type())
        h.instruments[metricName] = inst
    }
    h.mu.Unlock()
}
```

**Why**: OpenTelemetry instruments are expensive to create but cheap to reuse. This pattern:

- **Minimizes lock contention** in the hot path (metric recording uses fast read locks)
- **Ensures thread-safety** during instrument creation (write locks only when needed)
- **Scales well** under concurrent load (multiple goroutines can look up instruments simultaneously)

The double-check pattern prevents duplicate instrument creation when multiple goroutines race to create the same instrument for the first time.

### 4. Attribute Handling

**Implementation**: Direct conversion from `stats.Tag` to `attribute.KeyValue`
```go
func (h *SDKHandler) tagsToAttributes(tags []stats.Tag) []attribute.KeyValue {
    attrs := make([]attribute.KeyValue, 0, len(tags))
    for _, tag := range tags {
        attrs = append(attrs, attribute.String(tag.Name, tag.Value))
    }
    return attrs
}
```

**Why**: Simple 1:1 mapping preserves all user-provided metadata without transformation.

### 5. Resource Detection

**Pattern**: Leverage official OpenTelemetry resource detectors
```go
resource.New(ctx,
    resource.WithDetectors(ec2.NewResourceDetector()),
    resource.WithFromEnv(),
    resource.WithHost(),
    resource.WithProcess(),
)
```

**Why**: Automatic detection of cloud provider, Kubernetes, host, and process metadata without manual configuration.

## Performance Considerations

### Instrument Reuse
- Instruments are created once and cached
- RWMutex allows concurrent reads (the common case)
- Write locks only taken during initial instrument creation

### Gauge Recording
- Zero additional memory overhead (uses native Float64Gauge)
- Direct recording with no delta calculation required
- Simple O(1) operation per gauge recording

### Batching and Export Strategy

**Decision**: Delegate all batching to OpenTelemetry SDK's `PeriodicReader`

**Implementation**: No custom buffering or batching logic in the handler
```go
provider := sdkmetric.NewMeterProvider(
    sdkmetric.WithResource(res),
    sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
        sdkmetric.WithInterval(config.ExportInterval),  // Default: 10s
        sdkmetric.WithTimeout(config.ExportTimeout),    // Default: 30s
    )),
)
```

**Why**:
- The OTel SDK provides production-ready batching with in-memory aggregation
- `PeriodicReader` handles timing, aggregation reset, and export lifecycle
- Avoids reinventing batching logic and potential bugs
- Provides standard OTel behavior that users expect

**How it works**:
1. Metrics are recorded immediately to OTel instruments (no blocking)
2. SDK aggregates metrics in memory (e.g., summing counters, collecting histogram samples)
3. Every `ExportInterval`, the reader exports aggregated data and resets aggregations
4. Reduces network overhead and collector load automatically

**Trade-offs**:
- Metrics are not real-time (delayed by up to `ExportInterval`)
- Memory grows proportionally to metric cardinality until export
- Users must call `Flush()` before shutdown to export remaining metrics

## Error Handling

### Instrument Creation Failures
- Logged but don't block other metrics
- Silent no-op if instrument is nil
- Prevents cascade failures

### Export Failures
- Logged but don't stop metric collection
- Retries handled by OpenTelemetry SDK exporters
- Backoff and timeout configured at SDK level

### Context Cancellation
- Metric recording uses background context
- Unaffected by user context cancellation
- Shutdown still respects user-provided context

## Testing Strategy

### Unit Tests
- Instrument creation and caching
- Gauge delta calculation
- Value type conversions
- Protocol selection (HTTP vs gRPC)

### Integration Tests
- Environment variable configuration
- Multiple concurrent metrics
- Gauge absolute value semantics

### Benchmarks
- Metric recording performance
- Lock contention under load

## Limitations and Known Issues

### 1. No Exemplars
- Current implementation doesn't support exemplars
- Could be added in future versions

### 2. No Custom Views for Explicit Bucket Histograms
- Uses default aggregation and bucket boundaries for explicit bucket histograms
- Advanced users may want custom histogram buckets when not using exponential histograms

## Histogram Aggregation

### Exponential Histogram Support

**Implementation**: Configurable via `ExponentialHistogram` flag and View configuration

```go
if config.ExponentialHistogram {
    view := sdkmetric.NewView(
        sdkmetric.Instrument{Kind: sdkmetric.InstrumentKindHistogram},
        sdkmetric.Stream{
            Aggregation: sdkmetric.AggregationBase2ExponentialHistogram{
                MaxSize:  config.ExponentialHistogramMaxSize,  // Default: 160
                MaxScale: config.ExponentialHistogramMaxScale, // Default: 20
            },
        },
    )
    providerOpts = append(providerOpts, sdkmetric.WithView(view))
}
```

**Benefits**:
- **Better accuracy**: Adaptive buckets provide consistent relative error across value ranges
- **Lower memory**: Base-2 exponential buckets vs fixed explicit buckets
- **No pre-configuration**: Buckets adjust automatically to observed values
- **Modern standard**: Native support in Prometheus, Grafana, and OTLP backends

**How it works**:
1. Uses base-2 exponential buckets (powers of 2)
2. Automatically scales to accommodate value range
3. MaxSize limits total buckets (trades accuracy for memory)
4. MaxScale controls granularity (-10 to 20, where 20 = finest)

**Trade-offs**:
- Requires backend support (Prometheus 2.40+, modern OTLP collectors)
- Slightly higher CPU overhead during aggregation
- Not compatible with legacy systems expecting explicit buckets

**Default behavior**: When disabled, uses explicit bucket histogram with default boundaries

## Temporality Configuration

### What is Temporality?

Temporality determines whether metrics are reported as **cumulative totals** (since application start) or **deltas** (change since last export).

**Example - Request Counter**:
- **Cumulative**: Export "1000 total requests" → "1150 total requests" → "1320 total requests"
- **Delta**: Export "1000 new requests" → "150 new requests" → "170 new requests"

### Why We Use Cumulative Temporality (Default)

This handler uses **cumulative temporality** for all metrics by default. Here's why:

#### Compatibility with Prometheus and Standard Backends

- Prometheus expects cumulative counters and will graph them correctly
- Most OTLP backends (Grafana, Datadog, etc.) work best with cumulative data
- Industry standard practice in the OpenTelemetry ecosystem

#### Reliability and Query Simplicity

- **No data loss on export failures**: If an export fails, the next one still has complete data
- **Easier to query**: "How many total requests?" vs "Sum all deltas"
- **Converts to delta easily**: Backend can calculate rates from cumulative, but can't reconstruct cumulative from deltas

#### Lower Cognitive Load

- Counters show totals since start - intuitive and matches mental model
- Histograms show full distribution of all observations

### How It Works

**Cumulative semantics by instrument type**:

- **Counter** (`stats.Incr`, `stats.Add`): Total count since application start
  - Example: `requests.total` reports 1000, then 1150, then 1320

- **Histogram** (`stats.Observe`): Cumulative distribution of all observed values
  - Example: Latency histogram includes all requests since start

- **Gauge** (`stats.Set`): Current absolute value (temporality doesn't apply)
  - Example: `memory.used` always reports current memory usage

### Trade-offs

#### Advantages of Cumulative

- ✅ Prometheus and Grafana work out-of-box
- ✅ Resilient to export failures (no data loss)
- ✅ Backend can derive rates automatically
- ✅ Simpler mental model for most users

#### Disadvantages of Cumulative

- ❌ Slightly higher memory usage for high-cardinality counters
- ❌ Backend must calculate deltas for rate queries (minor overhead)
- ❌ Some specialized telemetry systems expect delta temporality

#### When Delta Might Be Better

- Your backend explicitly requires delta temporality (check docs)
- Extreme cardinality where cumulative memory overhead matters
- Building a custom metrics pipeline optimized for deltas

### Changing Temporality (Advanced)

If you need delta temporality, you can override the default:

```go
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    EndpointURL: "http://localhost:4317",
    // Use delta temporality for all instruments
    TemporalitySelector: sdkmetric.DeltaTemporalitySelector,
})
```

**Available selectors**:

- `sdkmetric.DefaultTemporalitySelector` - Cumulative for all (default, recommended)
- `sdkmetric.CumulativeTemporalitySelector` - Cumulative for all (explicit)
- `sdkmetric.DeltaTemporalitySelector` - Delta for all
- `sdkmetric.LowMemoryTemporalitySelector` - Delta for Counters/Histograms, Cumulative for UpDownCounters

**⚠️ Warning**: Changing temporality requires updating your backend configuration and queries. Most users should stick with the default cumulative temporality.

## Future Enhancements

### Potential Improvements
1. **Memory Management**: Add LRU eviction for unused instruments
2. **Exemplar Support**: Bridge to trace context for exemplars
3. **Custom Histogram Buckets**: Allow users to configure explicit bucket boundaries
4. **Metric Metadata**: Expose units and descriptions via OTel API

### OpenTelemetry SDK Evolution
- **Protocol Extensions**: Support new OTLP features as they're added
- **New Instrument Types**: Adopt new instrument types as they become available

## Migration from Legacy Handler

The legacy `Handler` in this package is marked as Alpha and has limitations:

**Legacy Handler Issues:**
- Custom OTLP implementation (not using official SDK)
- Only HTTP transport (despite having gRPC dependencies)
- No environment variable support
- No resource detection

**SDKHandler Advantages:**
- Official OpenTelemetry SDK
- Both HTTP and gRPC
- Full environment variable support
- Automatic resource detection
- Production-ready and well-tested

**Migration Path:**
```go
// Old (legacy)
handler := &otlp.Handler{
    Client: otlp.NewHTTPClient(endpoint),
    // ...
}

// New (recommended)
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolHTTPProtobuf,
    Endpoint: endpoint,
})
```

## References

- [OpenTelemetry Metrics Specification](https://opentelemetry.io/docs/specs/otel/metrics/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Go SDK Documentation](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/metric)
- [Resource Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/resource/)
