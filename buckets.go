package stats

import "strings"

// Key is a type used to uniquely identify metrics.
type Key struct {
	Measure string
	Field   string
}

// HistogramBuckets is a map type storing histogram buckets.
type HistogramBuckets map[Key][]Value

// Set sets a set of buckets to the given list of sorted values.
func (b HistogramBuckets) Set(key string, buckets ...any) {
	v := make([]Value, len(buckets))

	for i, b := range buckets {
		v[i] = MustValueOf(ValueOf(b))
	}

	b[makeKey(key)] = v
}

// SetKey registers buckets under an exact key.
//
// Set cannot express every key. It splits its argument on the last ".", so a
// field whose own name contains a "." is unreachable through it — and measures
// reported from struct tags routinely have such fields, "body.bytes" among
// them. SetKey takes the two halves directly.
func (b HistogramBuckets) SetKey(key Key, buckets ...any) {
	v := make([]Value, len(buckets))

	for i, x := range buckets {
		v[i] = MustValueOf(ValueOf(x))
	}

	b[key] = v
}

// SetUnprefixed registers buckets for a measure named without the prefix an
// engine will add to it.
//
// A package that instruments something — httpstats, netstats — registers its
// buckets from init(), before any engine exists, so it cannot know the prefix
// the measures it describes will end up carrying. Lookup resolves these by
// dropping leading segments from the measure name, so a set registered for
// "http.message" applies to "myapp.http.message" as well.
//
// Prefer SetKey wherever the full measure name is known. Matching by suffix
// cannot tell a derived name from an unrelated one that happens to end the
// same way, which is why it is opted into here rather than applied to every
// registration.
func (b HistogramBuckets) SetUnprefixed(key Key, buckets ...any) {
	b.SetKey(Key{Measure: anyPrefix + key.Measure, Field: key.Field}, buckets...)
}

// Lookup returns the buckets registered for a measure and field, or nil.
//
// An exact registration always wins. Failing that, registrations made with
// SetUnprefixed are tried against progressively shorter suffixes of the
// measure name, longest first.
func (b HistogramBuckets) Lookup(measure, field string) []Value {
	if v := b[Key{Measure: measure, Field: field}]; len(v) != 0 {
		return v
	}

	for {
		if v := b[Key{Measure: anyPrefix + measure, Field: field}]; len(v) != 0 {
			return v
		}

		i := strings.IndexByte(measure, '.')
		if i < 0 {
			return nil
		}

		measure = measure[i+1:]
	}
}

// anyPrefix marks a registration made by SetUnprefixed. It is not a valid
// measure name, so it cannot collide with one registered through Set or
// SetKey.
const anyPrefix = "*."

// Buckets is a registry where histogram buckets are placed. Some metric
// collection backends need to have histogram buckets defined by the program
// (like Prometheus), a common pattern is to use the init function of a package
// to register buckets for the various histograms that it produces.
var Buckets = HistogramBuckets{}

func makeKey(s string) Key {
	measure, field := splitMeasureField(s)
	return Key{Measure: measure, Field: field}
}

func splitMeasureField(s string) (measure, field string) {
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		measure, field = s[:i], s[i+1:]
	} else {
		field = s
	}
	return measure, field
}
