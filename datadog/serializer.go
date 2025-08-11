package datadog

import (
	"bytes"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	stats "github.com/segmentio/stats/v5"
)

// Datagram format: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell

type serializer struct {
	conn             io.WriteCloser
	bufferSize       int
	filters          map[string]struct{}
	distPrefixes     []string
	useDistributions bool
}

func (s *serializer) Write(b []byte) (int, error) {
	if s.conn == nil {
		return 0, io.ErrClosedPipe
	}

	// Ensure the serialized metric payload has valid UTF-8 encoded bytes.
	// Because ToValidUTF8 makes a copy make one pass through to ensure we
	// actually need to change anything.
	if !utf8.Valid(b) {
		b = bytes.ToValidUTF8(b, []byte("\uFFFD"))
	}
	if len(b) <= s.bufferSize {
		return s.conn.Write(b)
	}

	// When the serialized metrics are larger than the configured socket buffer
	// size, try to split them on '\n' characters.
	var n int

	for len(b) != 0 {
		var splitIndex int

		for splitIndex != len(b) {
			i := bytes.IndexByte(b[splitIndex:], '\n')
			if i < 0 {
				panic("stats/datadog: metrics are not formatted for the dogstatsd protocol")
			}
			if (i + splitIndex) >= s.bufferSize {
				if splitIndex == 0 {
					log.Printf("stats/datadog: metric of length %d B doesn't fit in the socket buffer of size %d B: %s", i+1, s.bufferSize, string(b))
					b = b[i+1:]
					continue
				}
				break
			}
			splitIndex += i + 1
		}

		c, err := s.conn.Write(b[:splitIndex])
		if err != nil {
			return n + c, err
		}

		n += c
		b = b[splitIndex:]
	}

	return n, nil
}

func (s *serializer) close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *serializer) AppendMeasures(b []byte, _ time.Time, measures ...stats.Measure) []byte {
	for _, m := range measures {
		b = s.AppendMeasure(b, m)
	}
	return b
}

var accentMap [256]byte

// valid[byte] = 1 if the ASCII char is allowed, 0 otherwise.
var valid = [128]bool{
	'.': true, '-': true, '_': true,
}

func init() {
	// Initialize all to identity mapping
	for i := range accentMap {
		accentMap[i] = byte(i)
	}

	// Latin-1 Supplement mappings (0xC0-0xFF)
	// Uppercase A variants
	accentMap[0xC0] = 'A' // À
	accentMap[0xC1] = 'A' // Á
	accentMap[0xC2] = 'A' // Â
	accentMap[0xC3] = 'A' // Ã
	accentMap[0xC4] = 'A' // Ä
	accentMap[0xC5] = 'A' // Å
	accentMap[0xC6] = 'A' // Æ -> A (could be "AE" but single char is simpler)

	// Uppercase C
	accentMap[0xC7] = 'C' // Ç

	// Uppercase E variants
	accentMap[0xC8] = 'E' // È
	accentMap[0xC9] = 'E' // É
	accentMap[0xCA] = 'E' // Ê
	accentMap[0xCB] = 'E' // Ë

	// Uppercase I variants
	accentMap[0xCC] = 'I' // Ì
	accentMap[0xCD] = 'I' // Í
	accentMap[0xCE] = 'I' // Î
	accentMap[0xCF] = 'I' // Ï

	// Uppercase D, N
	accentMap[0xD0] = 'D' // Ð
	accentMap[0xD1] = 'N' // Ñ

	// Uppercase O variants
	accentMap[0xD2] = 'O' // Ò
	accentMap[0xD3] = 'O' // Ó
	accentMap[0xD4] = 'O' // Ô
	accentMap[0xD5] = 'O' // Õ
	accentMap[0xD6] = 'O' // Ö
	accentMap[0xD8] = 'O' // Ø

	// Uppercase U variants
	accentMap[0xD9] = 'U' // Ù
	accentMap[0xDA] = 'U' // Ú
	accentMap[0xDB] = 'U' // Û
	accentMap[0xDC] = 'U' // Ü

	// Uppercase Y
	accentMap[0xDD] = 'Y' // Ý
	accentMap[0xDE] = 'T' // Þ (Thorn)

	// Lowercase sharp s
	accentMap[0xDF] = 's' // ß

	// Lowercase a variants
	accentMap[0xE0] = 'a' // à
	accentMap[0xE1] = 'a' // á
	accentMap[0xE2] = 'a' // â
	accentMap[0xE3] = 'a' // ã
	accentMap[0xE4] = 'a' // ä
	accentMap[0xE5] = 'a' // å
	accentMap[0xE6] = 'a' // æ -> a (could be "ae" but single char is simpler)

	// Lowercase c
	accentMap[0xE7] = 'c' // ç

	// Lowercase e variants
	accentMap[0xE8] = 'e' // è
	accentMap[0xE9] = 'e' // é
	accentMap[0xEA] = 'e' // ê
	accentMap[0xEB] = 'e' // ë

	// Lowercase i variants
	accentMap[0xEC] = 'i' // ì
	accentMap[0xED] = 'i' // í
	accentMap[0xEE] = 'i' // î
	accentMap[0xEF] = 'i' // ï

	// Lowercase d, n
	accentMap[0xF0] = 'd' // ð
	accentMap[0xF1] = 'n' // ñ

	// Lowercase o variants
	accentMap[0xF2] = 'o' // ò
	accentMap[0xF3] = 'o' // ó
	accentMap[0xF4] = 'o' // ô
	accentMap[0xF5] = 'o' // õ
	accentMap[0xF6] = 'o' // ö
	accentMap[0xF8] = 'o' // ø

	// Lowercase u variants
	accentMap[0xF9] = 'u' // ù
	accentMap[0xFA] = 'u' // ú
	accentMap[0xFB] = 'u' // û
	accentMap[0xFC] = 'u' // ü

	// Lowercase y
	accentMap[0xFD] = 'y' // ý
	accentMap[0xFE] = 't' // þ (thorn)
	accentMap[0xFF] = 'y' // ÿ

	for c := '0'; c <= '9'; c++ {
		valid[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		valid[c] = true
	}
	for c := 'a'; c <= 'z'; c++ {
		valid[c] = true
	}
}

const (
	replacement = byte('_') // what we substitute bad chars with
	maxLen      = 250       // guard for the StatsD UDP packet size
)

var shouldTrim [256]bool = [256]bool{
	'.': true,
	'_': true,
	'-': true,
}

// appendSanitizedMetricName converts *any* string into something that StatsD / Graphite
// accepts without complaints.
func appendSanitizedMetricName(dst []byte, raw string) []byte {
	if raw == "" {
		if len(dst) == 0 {
			return append(dst, "_unnamed_"...)
		}
		return dst
	}
	orig := len(dst)

	// Pre-grow
	need := len(raw)
	if need > maxLen {
		need = maxLen
	}
	if cap(dst)-len(dst) < need {
		nd := make([]byte, len(dst), len(dst)+need)
		copy(nd, dst)
		dst = nd
	}

	n := len(raw)
	i := 0
	lastWasReplacement := false

	// Skip leading trim while building
	for i < n {
		c := raw[i]
		if !shouldTrim[c] {
			break
		}
		i++
	}

	for i < n && (len(dst)-orig) < maxLen {
		// Batch ASCII-valid run
		remaining := maxLen - (len(dst) - orig)
		j := i
		limit := i + remaining
		if limit > n {
			limit = n
		}
		for j < limit {
			c := raw[j]
			if c >= 128 || !valid[c] {
				break
			}
			j++
		}
		if j > i {
			dst = append(dst, raw[i:j]...)
			lastWasReplacement = false
			i = j
			continue
		}

		// 2-byte common accent folding
		c0 := raw[i]
		if c0 >= 0xC2 && c0 <= 0xC3 && i+1 < n {
			c1 := raw[i+1]
			if c1 >= 0x80 && c1 <= 0xBF {
				code := uint16(c0&0x1F)<<6 | uint16(c1&0x3F)
				if code >= 0xC0 && code <= 0xFF {
					mapped := accentMap[code]
					if valid[mapped] && (len(dst)-orig) < maxLen {
						dst = append(dst, mapped)
						lastWasReplacement = false
						i += 2
						continue
					}
				}
			}
		}

		// Replacement for everything else
		if !lastWasReplacement && len(dst) > orig && (len(dst)-orig) < maxLen {
			dst = append(dst, replacement)
			lastWasReplacement = true
		}
		i++
	}

	// Trim trailing '.' '_' '-'
	for l := len(dst); l > orig; {
		c := dst[l-1]
		if !shouldTrim[c] {
			break
		}
		l--
		dst = dst[:l]
	}

	if len(dst) == orig {
		return append(dst, "_truncated_"...)
	}
	return dst
}

// AppendMeasure is a formatting routine to append the dogstatsd protocol
// representation of a measure to a memory buffer.
// Tags listed in the s.filters are removed. (some tags may not be suitable for submission to DataDog)
// Histogram metrics will be sent as distribution type if the metric name matches s.distPrefixes
// DogStatsd Protocol Docs: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell?tab=metrics#the-dogstatsd-protocol
func (s *serializer) AppendMeasure(b []byte, m stats.Measure) []byte {
	for _, field := range m.Fields {
		b = appendSanitizedMetricName(b, m.Name)
		if len(field.Name) > 0 {
			b = append(b, '.')
			b = appendSanitizedMetricName(b, field.Name)
		}

		b = append(b, ':')
		switch v := field.Value; v.Type() {
		case stats.Bool:
			if v.Bool() {
				b = append(b, '1')
			} else {
				b = append(b, '0')
			}
		case stats.Int:
			b = strconv.AppendInt(b, v.Int(), 10)
		case stats.Uint:
			b = strconv.AppendUint(b, v.Uint(), 10)
		case stats.Float:
			b = strconv.AppendFloat(b, normalizeFloat(v.Float()), 'g', -1, 64)
		case stats.Duration:
			b = strconv.AppendFloat(b, v.Duration().Seconds(), 'g', -1, 64)
		default:
			b = append(b, '0')
		}

		switch field.Type() {
		case stats.Counter:
			b = append(b, '|', 'c')
		case stats.Gauge:
			b = append(b, '|', 'g')
		default:
			if s.sendDist(field.Name) {
				b = append(b, '|', 'd')
			} else {
				b = append(b, '|', 'h')
			}
		}
		if len(m.Tags) > 0 {
			b = append(b, '|', '#')
			for i, t := range m.Tags {
				if _, skip := s.filters[t.Name]; skip {
					continue
				}
				if i != 0 {
					b = append(b, ',')
				}
				b = appendSanitizedMetricName(b, t.Name)
				b = append(b, ':')
				b = appendSanitizedMetricName(b, t.Value)
			}
		}
		b = append(b, '\n')
	}

	return b
}

// sendDist determines whether to send a metric to datadog as histogram `h` type or
// distribution `d` type. It's a confusing setup because useDistributions and distPrefixes
// are independent implementations of a control mechanism for sending distributions that
// aren't elegantly coordinated.
func (s *serializer) sendDist(name string) bool {
	if s.useDistributions {
		return true
	}

	if s.distPrefixes == nil {
		return false
	}
	for _, prefix := range s.distPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func normalizeFloat(f float64) float64 {
	switch {
	case math.IsNaN(f):
		return 0.0
	case math.IsInf(f, +1):
		return +math.MaxFloat64
	case math.IsInf(f, -1):
		return -math.MaxFloat64
	default:
		return f
	}
}
