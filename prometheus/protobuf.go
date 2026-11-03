package prometheus

import (
	"cmp"
	"io"
	"math"
	"mime"
	"slices"
	"strconv"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// protobufContentType is the Prometheus protobuf exposition: a stream of
// length-delimited io.prometheus.client.MetricFamily messages.
const protobufContentType = "application/vnd.google.protobuf; proto=io.prometheus.client.MetricFamily; encoding=delimited"

// acceptsProtobuf reports whether the Accept header prefers the protobuf
// exposition over the text format.
//
// Protobuf is served only when the scraper ranks it above the text format.
// Those are the only two formats the handler serves, so other types, such as
// OpenMetrics, are not ranked. Prometheus and Alloy list their scrape protocols with descending q
// values. A scraper puts protobuf first when its scrape protocols are set that
// way, and Prometheus also does so by default when native histograms or
// created-timestamp zero ingestion are on. Any other scraper keeps receiving
// text. Equal q values keep header order.
func acceptsProtobuf(accept string) bool {
	best, isProto := 0.0, false

	for part := range strings.SplitSeq(accept, ",") {
		mediatype, params, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err != nil {
			continue
		}

		q := 1.0
		if s, ok := params["q"]; ok {
			if q, err = strconv.ParseFloat(s, 64); err != nil {
				continue
			}
		}

		var pb bool
		switch mediatype {
		case "application/vnd.google.protobuf":
			if params["proto"] != "io.prometheus.client.MetricFamily" || params["encoding"] != "delimited" {
				continue
			}
			pb = true
		case "text/plain", "text/*", "*/*":
		default:
			continue
		}

		if q > best {
			best, isProto = q, pb
		}
	}

	return isProto
}

// writeProtobuf writes every metric as a length-delimited MetricFamily.
//
// Counters and histograms carry created_timestamp, the time their series
// started from zero. A scraper with start-timestamp zero ingestion enabled
// writes a zero sample at that time, so the first value of a new series is
// counted by increase() instead of being taken as its baseline.
//
// Names and labels are rendered exactly as WriteStats renders them, so
// switching format does not change any series.
func (h *Handler) writeProtobuf(w io.Writer) error {
	for _, mf := range h.metrics.families() {
		if _, err := protodelim.MarshalTo(w, mf); err != nil {
			return err
		}
	}
	return nil
}

type protoFamilyKey struct {
	name  string
	mtype metricType
}

type protoSeries struct {
	labels labels
	metric *dto.Metric
}

type protoFamily struct {
	family *dto.MetricFamily
	series []protoSeries
}

func (store *metricStore) families() []*dto.MetricFamily {
	byKey := make(map[protoFamilyKey]*protoFamily)

	store.mutex.RLock()

	for key, entry := range store.entries {
		var mtype dto.MetricType
		switch entry.mtype {
		case counter:
			mtype = dto.MetricType_COUNTER
		case gauge:
			mtype = dto.MetricType_GAUGE
		case histogram:
			mtype = dto.MetricType_HISTOGRAM
		default:
			// WriteStats publishes only these three types.
			continue
		}

		// Two entries can render to the same name, a.b and a_b for example.
		// The text format declares the family once and lists both under it,
		// so they are merged here too.
		name := string(appendMetricScopedName(nil, entry.scope, store.exposedName(key, entry)))
		fk := protoFamilyKey{name: name, mtype: entry.mtype}

		f := byKey[fk]
		if f == nil {
			f = &protoFamily{family: &dto.MetricFamily{Name: proto.String(name), Type: mtype.Enum()}}
			byKey[fk] = f
		}

		entry.mutex.RLock()
		for _, states := range entry.states {
			for _, state := range states {
				f.series = append(f.series, protoSeries{
					labels: state.labels,
					metric: state.protoMetric(entry.mtype),
				})
			}
		}
		entry.mutex.RUnlock()
	}

	store.mutex.RUnlock()

	families := make([]*dto.MetricFamily, 0, len(byKey))
	for _, f := range byKey {
		if len(f.series) == 0 {
			continue
		}
		slices.SortFunc(f.series, func(a, b protoSeries) int {
			switch {
			case a.labels.less(b.labels):
				return -1
			case b.labels.less(a.labels):
				return 1
			}
			return 0
		})
		for _, s := range f.series {
			f.family.Metric = append(f.family.Metric, s.metric)
		}
		families = append(families, f.family)
	}

	slices.SortFunc(families, func(a, b *dto.MetricFamily) int {
		return cmp.Compare(a.GetName(), b.GetName())
	})
	return families
}

func (state *metricState) protoMetric(mtype metricType) *dto.Metric {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	m := &dto.Metric{Label: protoLabels(state.labels)}

	switch mtype {
	case counter:
		m.Counter = &dto.Counter{
			Value:            proto.Float64(state.value),
			CreatedTimestamp: protoCreated(state),
		}

	case gauge:
		m.Gauge = &dto.Gauge{Value: proto.Float64(state.value)}

	case histogram:
		// The +Inf bucket is left out, as client_golang does: its count is
		// sample_count, and the scraper adds the le="+Inf" series itself.
		h := &dto.Histogram{
			SampleCount:      proto.Uint64(state.count),
			SampleSum:        proto.Float64(state.sum),
			CreatedTimestamp: protoCreated(state),
		}
		var cumulativeCount uint64
		for _, bucket := range state.buckets {
			cumulativeCount += bucket.count
			if math.IsInf(bucket.limit, 1) {
				continue
			}
			h.Bucket = append(h.Bucket, &dto.Bucket{
				CumulativeCount: proto.Uint64(cumulativeCount),
				UpperBound:      proto.Float64(bucket.limit),
			})
		}
		m.Histogram = h
	}

	return m
}

func protoCreated(state *metricState) *timestamppb.Timestamp {
	if state.created.IsZero() {
		return nil
	}
	return timestamppb.New(state.created)
}

func protoLabels(l labels) []*dto.LabelPair {
	if len(l) == 0 {
		return nil
	}
	pairs := make([]*dto.LabelPair, len(l))
	for i, label := range l {
		pairs[i] = &dto.LabelPair{
			Name:  proto.String(string(appendLabelName(nil, label.name))),
			Value: proto.String(label.value),
		}
	}
	return pairs
}
