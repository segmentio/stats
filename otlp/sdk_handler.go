package otlp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/segmentio/stats/v5"
)

// Protocol defines the transport protocol for OTLP export
type Protocol string

const (
	// ProtocolGRPC uses gRPC transport
	ProtocolGRPC Protocol = "grpc"
	// ProtocolHTTPProtobuf uses HTTP with protobuf encoding
	ProtocolHTTPProtobuf Protocol = "http/protobuf"
)

// SDKHandler implements stats.Handler using the official OpenTelemetry SDK.
// It bridges stats metrics to OTel metrics and supports both HTTP and gRPC transports.
//
// This handler supports all standard OpenTelemetry environment variables:
//   - OTEL_EXPORTER_OTLP_ENDPOINT
//   - OTEL_EXPORTER_OTLP_PROTOCOL (grpc, http/protobuf)
//   - OTEL_EXPORTER_OTLP_HEADERS
//   - OTEL_EXPORTER_OTLP_TIMEOUT
//   - OTEL_RESOURCE_ATTRIBUTES
//   - OTEL_SERVICE_NAME
//   - And more...
//
// Example usage:
//
//	handler, err := otlp.NewSDKHandler(ctx, otlp.SDKConfig{
//	    Protocol: otlp.ProtocolGRPC,
//	    Endpoint: "localhost:4317",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer handler.Shutdown(ctx)
//	stats.Register(handler)
type SDKHandler struct {
	provider      *sdkmetric.MeterProvider
	shutdownCtx   context.Context // Context for shutdown operations only
	mu            sync.RWMutex
	instruments   map[string]instrument
	resourceAttrs []attribute.KeyValue
}

type instrument struct {
	counter   otelmetric.Int64Counter
	gauge     otelmetric.Float64Gauge
	histogram otelmetric.Float64Histogram
}

// SDKConfig contains configuration for the OpenTelemetry SDK handler
type SDKConfig struct {
	// Protocol specifies the transport protocol (grpc or http/protobuf)
	// If empty, defaults to ProtocolGRPC
	Protocol Protocol

	// Endpoint specifies the OTLP endpoint
	// For gRPC: "localhost:4317"
	// For HTTP: "http://localhost:4318"
	// If empty, uses OTEL_EXPORTER_OTLP_ENDPOINT environment variable
	Endpoint string

	// Resource specifies the resource attributes for all metrics
	// If nil, uses automatic resource detection
	Resource *resource.Resource

	// ExportInterval specifies how often to export metrics
	// If zero, defaults to 10 seconds
	ExportInterval time.Duration

	// ExportTimeout specifies the timeout for exports
	// If zero, defaults to 30 seconds
	ExportTimeout time.Duration

	// HTTPOptions are additional options for HTTP protocol
	// Only used when Protocol is ProtocolHTTPProtobuf
	HTTPOptions []otlpmetrichttp.Option

	// GRPCOptions are additional options for gRPC protocol
	// Only used when Protocol is ProtocolGRPC
	GRPCOptions []otlpmetricgrpc.Option

	// ExponentialHistogram enables exponential histogram aggregation for histogram metrics.
	// When true, histograms use base-2 exponential buckets which provide better accuracy
	// and lower memory overhead compared to explicit bucket histograms.
	// Default: false (uses explicit bucket histograms)
	ExponentialHistogram bool

	// ExponentialHistogramMaxSize sets the maximum number of buckets for exponential histograms.
	// Larger values provide better accuracy but use more memory.
	// Default: 160 (if ExponentialHistogram is true)
	// Ignored if ExponentialHistogram is false
	ExponentialHistogramMaxSize int32

	// ExponentialHistogramMaxScale sets the maximum scale (resolution) for exponential histograms.
	// Higher values provide finer bucket granularity.
	// Valid range: -10 to 20
	// Default: 20 (if ExponentialHistogram is true)
	// Ignored if ExponentialHistogram is false
	ExponentialHistogramMaxScale int32

	// TemporalitySelector determines the temporality (cumulative vs delta) for each instrument kind.
	// If nil, uses DefaultTemporalitySelector which returns CumulativeTemporality for all instruments.
	// This is recommended for Prometheus and most OTLP backends.
	//
	// Available selectors:
	//   - sdkmetric.DefaultTemporalitySelector: Cumulative for all (default, Prometheus-compatible)
	//   - sdkmetric.CumulativeTemporalitySelector: Cumulative for all
	//   - sdkmetric.DeltaTemporalitySelector: Delta for all
	//   - sdkmetric.LowMemoryTemporalitySelector: Delta for Counters/Histograms, Cumulative for UpDownCounters
	TemporalitySelector sdkmetric.TemporalitySelector
}

// NewSDKHandler creates a new handler using the OpenTelemetry SDK.
// It automatically detects resources and supports all standard OTEL environment variables.
func NewSDKHandler(ctx context.Context, config SDKConfig) (*SDKHandler, error) {
	// Set defaults
	if config.Protocol == "" {
		config.Protocol = ProtocolGRPC
	}
	if config.ExportInterval == 0 {
		config.ExportInterval = 10 * time.Second
	}
	if config.ExportTimeout == 0 {
		config.ExportTimeout = 30 * time.Second
	}
	if config.ExponentialHistogram {
		if config.ExponentialHistogramMaxSize == 0 {
			config.ExponentialHistogramMaxSize = 160
		}
		if config.ExponentialHistogramMaxScale == 0 {
			config.ExponentialHistogramMaxScale = 20
		}
	}

	// Create resource if not provided
	res := config.Resource
	if res == nil {
		var err error
		res, err = resource.New(ctx,
			resource.WithFromEnv(),
			resource.WithTelemetrySDK(),
			resource.WithHost(),
			resource.WithProcess(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}
	}

	// Create exporter based on protocol
	var exporter sdkmetric.Exporter
	var err error

	switch config.Protocol {
	case ProtocolGRPC:
		opts := config.GRPCOptions
		if config.Endpoint != "" {
			opts = append([]otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(config.Endpoint)}, opts...)
		}
		// Configure temporality if provided (default is cumulative, which is Prometheus-compatible)
		if config.TemporalitySelector != nil {
			opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(config.TemporalitySelector))
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC exporter: %w", err)
		}

	case ProtocolHTTPProtobuf:
		opts := config.HTTPOptions
		if config.Endpoint != "" {
			opts = append([]otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(config.Endpoint)}, opts...)
		}
		// Configure temporality if provided (default is cumulative, which is Prometheus-compatible)
		if config.TemporalitySelector != nil {
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(config.TemporalitySelector))
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP exporter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", config.Protocol)
	}

	// Configure histogram aggregation if exponential histograms are enabled
	var providerOpts []sdkmetric.Option
	providerOpts = append(providerOpts, sdkmetric.WithResource(res))
	providerOpts = append(providerOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(config.ExportInterval),
		sdkmetric.WithTimeout(config.ExportTimeout),
	)))

	if config.ExponentialHistogram {
		// Configure exponential histogram aggregation for all histogram instruments
		view := sdkmetric.NewView(
			sdkmetric.Instrument{Kind: sdkmetric.InstrumentKindHistogram},
			sdkmetric.Stream{
				Aggregation: sdkmetric.AggregationBase2ExponentialHistogram{
					MaxSize:  config.ExponentialHistogramMaxSize,
					MaxScale: config.ExponentialHistogramMaxScale,
				},
			},
		)
		providerOpts = append(providerOpts, sdkmetric.WithView(view))
	}

	// Create meter provider with configured options
	provider := sdkmetric.NewMeterProvider(providerOpts...)

	return &SDKHandler{
		provider:    provider,
		shutdownCtx: ctx,
		instruments: make(map[string]instrument),
	}, nil
}

// NewSDKHandlerFromEnv creates a handler using only environment variables.
// This is the simplest way to create a handler with full OpenTelemetry support.
//
// It respects all standard OTEL environment variables including:
//   - OTEL_EXPORTER_OTLP_ENDPOINT
//   - OTEL_EXPORTER_OTLP_PROTOCOL
//   - OTEL_EXPORTER_OTLP_HEADERS
//   - OTEL_RESOURCE_ATTRIBUTES
//   - OTEL_SERVICE_NAME
func NewSDKHandlerFromEnv(ctx context.Context) (*SDKHandler, error) {
	// The SDK exporters will automatically read all environment variables
	return NewSDKHandler(ctx, SDKConfig{
		Protocol: ProtocolGRPC, // Can be overridden by OTEL_EXPORTER_OTLP_PROTOCOL
	})
}

// HandleMeasures implements stats.Handler
func (h *SDKHandler) HandleMeasures(t time.Time, measures ...stats.Measure) {
	// Use background context for recording metrics to avoid context cancellation issues
	// The shutdownCtx is only used for shutdown operations
	ctx := context.Background()

	meter := h.provider.Meter("github.com/segmentio/stats")

	for _, measure := range measures {
		for _, field := range measure.Fields {
			metricName := measure.Name + "." + field.Name
			attrs := h.tagsToAttributes(measure.Tags)

			h.mu.RLock()
			inst, exists := h.instruments[metricName]
			h.mu.RUnlock()

			if !exists {
				h.mu.Lock()
				// Double-check after acquiring write lock
				inst, exists = h.instruments[metricName]
				if !exists {
					inst = h.createInstruments(meter, metricName, field.Type())
					h.instruments[metricName] = inst
				}
				h.mu.Unlock()
			}

			h.recordMetric(ctx, inst, field, metricName, attrs)
		}
	}
}

// createInstruments creates OTel instruments based on field type
func (h *SDKHandler) createInstruments(meter otelmetric.Meter, name string, fieldType stats.FieldType) instrument {
	var inst instrument

	switch fieldType {
	case stats.Counter:
		counter, err := meter.Int64Counter(name)
		if err != nil {
			log.Printf("stats/otlp: failed to create counter %s: %v", name, err)
		}
		inst.counter = counter

	case stats.Gauge:
		// Use Float64Gauge for gauges (synchronous gauge instrument)
		gauge, err := meter.Float64Gauge(name)
		if err != nil {
			log.Printf("stats/otlp: failed to create gauge %s: %v", name, err)
		}
		inst.gauge = gauge

	case stats.Histogram:
		histogram, err := meter.Float64Histogram(name)
		if err != nil {
			log.Printf("stats/otlp: failed to create histogram %s: %v", name, err)
		}
		inst.histogram = histogram
	}

	return inst
}

// recordMetric records a metric value to the appropriate instrument
func (h *SDKHandler) recordMetric(ctx context.Context, inst instrument, field stats.Field, metricName string, attrs []attribute.KeyValue) {
	opts := otelmetric.WithAttributes(attrs...)

	switch field.Type() {
	case stats.Counter:
		if inst.counter != nil {
			inst.counter.Add(ctx, h.valueToInt64(field.Value), opts)
		}

	case stats.Gauge:
		if inst.gauge != nil {
			// Gauges record instantaneous values directly
			inst.gauge.Record(ctx, h.valueToFloat64(field.Value), opts)
		}

	case stats.Histogram:
		if inst.histogram != nil {
			inst.histogram.Record(ctx, h.valueToFloat64(field.Value), opts)
		}
	}
}

// tagsToAttributes converts stats tags to OTel attributes
func (h *SDKHandler) tagsToAttributes(tags []stats.Tag) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(tags))
	for _, tag := range tags {
		attrs = append(attrs, attribute.String(tag.Name, tag.Value))
	}
	return attrs
}

// valueToInt64 converts stats.Value to int64 for counters
func (h *SDKHandler) valueToInt64(v stats.Value) int64 {
	switch v.Type() {
	case stats.Bool:
		if v.Bool() {
			return 1
		}
		return 0
	case stats.Int:
		return v.Int()
	case stats.Uint:
		return int64(v.Uint())
	case stats.Float:
		return int64(v.Float())
	case stats.Duration:
		return int64(v.Duration().Nanoseconds())
	}
	return 0
}

// valueToFloat64 converts stats.Value to float64 for gauges and histograms
func (h *SDKHandler) valueToFloat64(v stats.Value) float64 {
	switch v.Type() {
	case stats.Bool:
		if v.Bool() {
			return 1.0
		}
		return 0.0
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

// Flush implements stats.Flusher
func (h *SDKHandler) Flush() {
	if err := h.provider.ForceFlush(h.shutdownCtx); err != nil {
		log.Printf("stats/otlp: failed to flush: %v", err)
	}
}

// Shutdown gracefully shuts down the handler and exports any remaining metrics
func (h *SDKHandler) Shutdown(ctx context.Context) error {
	return h.provider.Shutdown(ctx)
}
