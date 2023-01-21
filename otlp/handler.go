package otlp

import (
	"container/list"
	"context"
	"fmt"
	"hash/maphash"
	"log"
	"sync"
	"time"

	"github.com/vertoforce/stats"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

const (
	// DefaultMaxMetrics is the default maximum of metrics kept in memory
	// by the handler.
	DefaultMaxMetrics = 5000

	// DefaultFlushInterval is the default interval to flush the metrics
	// to the OpenTelemetry destination.
	//
	// Metrics will be flushed to the destination when DefaultFlushInterval or
	// DefaultMaxMetrics are reached, whichever comes first.
	DefaultFlushInterval = 10 * time.Second
)

// Status: Alpha. This Handler is still in heavy development phase.
//
//	Do not use in production.
//
// Handler implements stats.Handler to be used to forward metrics to an
// OpenTelemetry destination. Usually an OpenTelemetry Collector.
//
// With the current implementation this Handler is targeting a Prometheus
// based backend or any backend expecting cumulative values.
//
// This Handler leverages a doubly linked list with a map to implement
// a ring buffer with a lookup to ensure a low memory usage.
type Handler struct {
	Client        Client
	Context       context.Context
	FlushInterval time.Duration
	MaxMetrics    int

	once sync.Once

	mu      sync.RWMutex
	ordered list.List
	metrics map[uint64]*list.Element
}

var (
	hashseed = maphash.MakeSeed()
)

// NewHandler return an instance of Handler with the default client,
// flush interval and in-memory metrics limit.
func NewHandler(ctx context.Context, endpoint string) *Handler {
	return &Handler{
		Client:        NewHTTPClient(endpoint),
		Context:       ctx,
		FlushInterval: DefaultFlushInterval,
		MaxMetrics:    DefaultMaxMetrics,
	}
}

func (h *Handler) HandlerMeasure(t time.Time, measures ...stats.Measure) {
	h.once.Do(func() {
		if h.FlushInterval == 0 {
			return
		}

		go h.start(h.Context)
	})

	h.handleMeasures(t, measures...)
}

func (h *Handler) start(ctx context.Context) {
	defer h.flush()

	t := time.NewTicker(h.FlushInterval)

	for {
		select {
		case <-t.C:
			if err := h.flush(); err != nil {
				log.Printf("stats/otlp: %s", err)
			}
		case <-ctx.Done():
			break
		}
	}
}

func (h *Handler) handleMeasures(t time.Time, measures ...stats.Measure) {
	for _, measure := range measures {
		for _, field := range measure.Fields {
			m := metric{
				time:        t,
				measureName: measure.Name,
				fieldName:   field.Name,
				fieldType:   field.Type(),
				tags:        measure.Tags,
				value:       field.Value,
			}

			if field.Type() == stats.Histogram {
				k := stats.Key{Measure: measure.Name, Field: field.Name}
				m.sum = valueOf(m.value)
				m.buckets = makeMetricBuckets(stats.Buckets[k])
				m.buckets.update(valueOf(m.value))
				m.count++
			}

			sign := m.signature()
			m.sign = sign

			known := h.lookup(sign, func(a *metric) *metric {
				switch a.fieldType {
				case stats.Counter:
					a.value = a.add(m.value)
				case stats.Histogram:
					a.sum += valueOf(m.value)
					a.count++
					for i := range a.buckets {
						a.buckets[i].count += m.buckets[i].count
					}
				}
				return a
			})

			if known == nil {
				n := h.push(sign, &m)
				if n > h.MaxMetrics {
					if err := h.flush(); err != nil {
						log.Printf("stats/otlp: %s", err)
					}
				}
			}
		}
	}
}

func (h *Handler) flush() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	metrics := []*metricpb.Metric{}

	for e := h.ordered.Front(); e != nil; e = e.Next() {
		m := e.Value.(*metric)
		if m.flushed {
			continue
		}
		metrics = append(metrics, convertMetrics(*m)...)
		m.flushed = true
	}

	if len(metrics) == 0 {
		return nil
	}

	//FIXME how big can a metrics service request be ? need pagination ?
	request := &colmetricpb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricpb.ResourceMetrics{
			{
				ScopeMetrics: []*metricpb.ScopeMetrics{
					{Metrics: metrics},
				},
			},
		},
	}

	if err := h.Client.Handle(h.Context, request); err != nil {
		return fmt.Errorf("failed to flush measures: %s", err)
	}

	return nil
}

func (h *Handler) lookup(signature uint64, update func(*metric) *metric) *metric {
	h.mu.Lock()
	defer h.mu.Unlock()

	if m := h.metrics[signature]; m != nil {
		h.ordered.MoveToFront(m)
		m.Value = update(m.Value.(*metric))
		return m.Value.(*metric)
	}

	return nil
}

func (h *Handler) push(sign uint64, m *metric) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.metrics == nil {
		h.metrics = map[uint64]*list.Element{}
	}

	element := h.ordered.PushFront(m)
	h.metrics[sign] = element

	if len(h.metrics) > h.MaxMetrics {
		last := h.ordered.Back()
		h.ordered.Remove(last)
		delete(h.metrics, last.Value.(*metric).sign)
	}

	return len(h.metrics)
}
