package prometheus

import (
	"cmp"
	"io"
	"math"
	"mime"
	"slices"
	"strconv"
	"strings"
	"time"

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
// switching format does not change any series on Prometheus 3.x. Histogram
// bounds are the exception on 2.x: the scraper formats UpperBound itself, as
// le="1.0" where the text format writes le="1", and only 3.x normalizes the
// text value to match.
//
// Each family's messages are built just before it is written and dropped
// after, so only one family's worth is held at a time.
func (h *Handler) writeProtobuf(w io.Writer) error {
	for _, f := range h.metrics.families() {
		if _, err := protodelim.MarshalTo(w, f.message()); err != nil {
			return err
		}
	}
	return nil
}

type protoFamilyKey struct {
	name  string
	mtype metricType
}

type protoFamily struct {
	name   string
	mtype  metricType
	series []protoSeries
	first  int // position of the family's first entry, for ordering
}

func (f *protoFamily) message() *dto.MetricFamily {
	var mtype dto.MetricType
	switch f.mtype {
	case counter:
		mtype = dto.MetricType_COUNTER
	case gauge:
		mtype = dto.MetricType_GAUGE
	case histogram:
		mtype = dto.MetricType_HISTOGRAM
	}

	mf := &dto.MetricFamily{
		Name:   proto.String(f.name),
		Type:   mtype.Enum(),
		Metric: make([]*dto.Metric, len(f.series)),
	}
	for i := range f.series {
		mf.Metric[i] = f.series[i].protoMetric(f.mtype)
	}
	return mf
}

// protoEntry is a store entry copied out under the locks, so that the
// messages can be built after they are released.
type protoEntry struct {
	scope  string
	name   string
	mtype  metricType
	series []protoSeries
}

type protoSeries struct {
	labels  labels
	value   float64
	sum     float64
	count   uint64
	created time.Time
	buckets []protoBucket
}

type protoBucket struct {
	limit float64
	count uint64
}

// families returns the families to publish in order, each with its series
// sorted, without building their messages.
func (store *metricStore) families() []*protoFamily {
	byKey := make(map[protoFamilyKey]*protoFamily)

	// Entries come out of a map, so they are put in the order WriteStats
	// uses before anything depends on it. Where two entries render to the
	// same name and labels, a scraper keeps the first sample and drops the
	// other: an order that changed between scrapes would make the stored
	// value alternate between two different series, and increase() would
	// read every switch as a reset.
	entries := store.protoEntries()
	slices.SortFunc(entries, func(a, b protoEntry) int {
		return cmp.Or(
			cmp.Compare(a.scope, b.scope),
			cmp.Compare(a.name, b.name),
			cmp.Compare(a.mtype, b.mtype),
		)
	})

	for i, entry := range entries {
		// Two entries can render to the same name, a.b and a_b for example.
		// Their series are merged into one family, so the name is declared
		// once.
		name := string(appendMetricScopedName(nil, entry.scope, entry.name))
		fk := protoFamilyKey{name: name, mtype: entry.mtype}

		f := byKey[fk]
		if f == nil {
			f = &protoFamily{name: name, mtype: entry.mtype, first: i}
			byKey[fk] = f
		}
		f.series = append(f.series, entry.series...)
	}

	families := make([]*protoFamily, 0, len(byKey))
	for _, f := range byKey {
		if len(f.series) == 0 {
			continue
		}
		// Stable, so that series with the same labels keep the order of the
		// entries they came from.
		slices.SortStableFunc(f.series, func(a, b protoSeries) int {
			switch {
			case a.labels.less(b.labels):
				return -1
			case b.labels.less(a.labels):
				return 1
			}
			return 0
		})
		families = append(families, f)
	}

	// A counter and a gauge can render to the same name too; their families
	// are ordered by their first entry.
	slices.SortFunc(families, func(a, b *protoFamily) int {
		return cmp.Or(
			cmp.Compare(a.name, b.name),
			cmp.Compare(a.first, b.first),
		)
	})
	return families
}

// protoEntries copies out every series the protobuf exposition publishes.
//
// The locks are held only for the copy, as collect holds them. A writer
// waiting on store.mutex blocks every reader queued behind it, HandleMeasures
// included, so building the messages under the read lock would stall the
// application for as long as the build takes.
func (store *metricStore) protoEntries() []protoEntry {
	// The bucket copies share one backing array, so that copying a histogram
	// does not cost an allocation per series.
	var buckets []protoBucket

	store.mutex.RLock()
	entries := make([]protoEntry, 0, len(store.entries))

	for key, entry := range store.entries {
		switch entry.mtype {
		case counter, gauge, histogram:
		default:
			// WriteStats publishes only these three types.
			continue
		}

		e := protoEntry{scope: entry.scope, name: store.exposedName(key, entry), mtype: entry.mtype}

		entry.mutex.RLock()
		for _, states := range entry.states {
			for _, state := range states {
				var series protoSeries
				series, buckets = state.protoSeries(buckets)
				e.series = append(e.series, series)
			}
		}
		entry.mutex.RUnlock()

		entries = append(entries, e)
	}

	store.mutex.RUnlock()
	return entries
}

// protoSeries copies the state, appending its buckets to arena.
func (state *metricState) protoSeries(arena []protoBucket) (protoSeries, []protoBucket) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	s := protoSeries{
		labels:  state.labels,
		value:   state.value,
		sum:     state.sum,
		count:   state.count,
		created: state.created,
	}
	if len(state.buckets) != 0 {
		start := len(arena)
		for _, bucket := range state.buckets {
			arena = append(arena, protoBucket{limit: bucket.limit, count: bucket.count})
		}
		s.buckets = arena[start:len(arena):len(arena)]
	}
	return s, arena
}

func (s *protoSeries) protoMetric(mtype metricType) *dto.Metric {
	m := &dto.Metric{Label: protoLabels(s.labels)}

	switch mtype {
	case counter:
		m.Counter = &dto.Counter{
			Value:            proto.Float64(s.value),
			CreatedTimestamp: protoCreated(s.created),
		}

	case gauge:
		m.Gauge = &dto.Gauge{Value: proto.Float64(s.value)}

	case histogram:
		// The +Inf bucket is left out, as client_golang does: its count is
		// sample_count, and the scraper adds the le="+Inf" series itself.
		h := &dto.Histogram{
			SampleCount:      proto.Uint64(s.count),
			SampleSum:        proto.Float64(s.sum),
			CreatedTimestamp: protoCreated(s.created),
		}
		var cumulativeCount uint64
		for _, bucket := range s.buckets {
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

func protoCreated(created time.Time) *timestamppb.Timestamp {
	if created.IsZero() {
		return nil
	}
	return timestamppb.New(created)
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
