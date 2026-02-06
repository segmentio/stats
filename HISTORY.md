# History

### v5.9.0 (February 6, 2026)

Add full OpenTelemetry OTLP exporter support with official SDK integration.

**New Feature: OpenTelemetry OTLP Exporter**

The `otlp` package now includes a production-ready `SDKHandler` that uses the
official OpenTelemetry SDK with comprehensive support for modern observability
requirements:

- **Dual Transport Support**: Both gRPC and HTTP/Protobuf protocols
- **Environment Variables**: Full support for all standard `OTEL_*` environment
  variables including `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_PROTOCOL`,
  `OTEL_RESOURCE_ATTRIBUTES`, etc.
- **Automatic Resource Detection**: Built-in support for AWS (EC2, ECS, EKS, Lambda),
  GCP (Compute Engine), Azure (VM), Kubernetes, host, and process metadata
- **All Metric Types**: Counter, Gauge, and Histogram with proper semantics
- **Tag Preservation**: Automatic conversion of stats tags to OpenTelemetry attributes
- **Production Ready**: Thread-safe instrument caching, proper context handling,
  and comprehensive error handling

**Usage Example:**

```go
import (
    "context"
    "github.com/segmentio/stats/v5"
    "github.com/segmentio/stats/v5/otlp"
)

// Simple usage with environment variables
handler, err := otlp.NewSDKHandlerFromEnv(ctx)
if err != nil {
    log.Fatal(err)
}
defer handler.Shutdown(ctx)
stats.Register(handler)

// Or with explicit configuration
handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
    Protocol: otlp.ProtocolGRPC,
    Endpoint: "localhost:4317",
})
```

**Implementation Details:**

- Gauges use `UpDownCounter` with delta calculation to maintain absolute value
  semantics (workaround until stable OTel SDK adds Gauge instrument)
- Background context for metric recording to prevent context cancellation issues
- Lock-free reads for instrument lookup in the hot path
- Comprehensive documentation including cloud resource detector examples

See the [otlp package documentation](./otlp/README.md) for complete details and examples.

### v5.8.0 (December 15, 2025)

When reporting go/stats versions, ensure that any user provided tags are
included with the go-version and stats version reporting tags, to ensure
better correlation on the Datadog side.

At the same time, don't report a timestamp with this metric, to avoid problems
where Prometheus says that the metric is too old (we only report it one time).

### v5.7.0 (November 24, 2025)

More lenient sanitization for tag values, which can contain commas, slashes, and
other characters that are not allowed in a metric name.

### v5.6.5 (October 14, 2025)

The sanitization process introduced in the v5.6.0 release did not properly
handle characters in the extended Latin-1 supplement, e.g. "÷". This issue has
been fixed in this release.

### v5.6.4 (September 16, 2025)

Remove golang.org/x/exp from the list of dependencies, in favor of the "slices"
standard library package. This bumps the minimum supported Go version to 1.23.

### v5.6.3 (August 29, 2025)

The Datadog client should have faster performance, by copying less metric data
before writing it to the socket.

### v5.6.2 (July 10, 2025)

Remove outdated README content and add debugstats examples.

### v5.6.1 (May 29, 2025)

Fix an error in the v5.6.0 release related to metric names with a longer buffer
size.

### v5.6.0 (May 27, 2025)

- In the `datadog` library: invalid characters in metric names, field
names, or tag keys/values will be replaced with underscores. Accents and
other diacritics will be removed (e.g. é will be replaced with 'e').
This change also improves performance of HandleMeasures by about 15-20%.
[#192](https://github.com/segmentio/stats/pull/192)

- External calls to github.com/segmentio/objconv were replaced by imports of
  github.com/segmentio/stats/v5/util/objconv, which is a fork of the library (it
  has since been archived). This allowed to to substantially reduce the surface
  we import.

    - `influxdb`: calls to objconv/json were replaced with
    github.com/segmentio/encoding/json (a library with
    substantially more production experience and test coverage).
    [#193](https://github.com/segmentio/stats/pull/193)

- `prometheus`: Fix a deadlock from concurrent calls to collect() and cleanup().
  Thank you Matthew Hooker for this contribution.
  [#194](https://github.com/segmentio/stats/pull/194)

### v5.5.0 (March 26, 2025)

- Add logic to replace invalid unicode in the serialized datadog payload with '\ufffd'.

### v5.4.0 (February 21, 2025)

- Fix a regression in configured buffer size for the datadog client. Versions
5.0.0 to 5.3.1 would ignore any configured buffer size and use the default value
of 1024. This could lead to smaller than expected writes and contention for the
file handle.

### v5.3.1 (January 2, 2025)

- Fix version parsing logic.

### v5.3.0 (December 19, 2024)

- Add `debugstats` package that can easily print metrics to stdout.

- Fix file handle leak in procstats in the DelayMetrics collector.

### v5.2.0

- `go_version.value` and `stats_version.value` will be emitted the first
time any metrics are sent from an Engine. Disable this behavior by setting
`GoVersionReportingEnabled = false` in code, or setting environment variable
`STATS_DISABLE_GO_VERSION_REPORTING=true`.

### v5.1.0

Add support for publishing stats to Unix datagram sockets (UDS).

### v5.0.0 (Released on September 11, 2024)

In the `httpstats` package, replace misspelled `http_req_content_endoing`
and `http_res_content_endoing` with `http_req_content_encoding` and
`http_res_content_encoding`, respectively. This is a breaking change; any
dashboards or queries that filter on this tag must be updated.
