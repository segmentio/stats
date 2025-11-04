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

	// Ensure the serialized metric payload has valid UTF-8 encoded bytes
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

// latin1SupplementMap maps Unicode codepoints U+00C0-U+00FF (Latin-1 Supplement)
// to their unaccented ASCII equivalents. This is used to handle common accented
// characters in metric names.
//
// Note: This array is indexed by codepoint values (e.g., U+00E9 for é), which
// numerically match the byte values in the Latin-1 encoding. The mapping handles
// 2-byte UTF-8 sequences that decode to these codepoints.
var latin1SupplementMap [256]byte

// valid[byte] = 1 if the ASCII char is allowed, 0 otherwise.
var valid = [256]bool{
	'.': true, '-': true, '_': true,
}

func init() {
	// Initialize all to identity mapping
	for i := range latin1SupplementMap {
		latin1SupplementMap[i] = byte(i)
	}

	// Latin-1 Supplement mappings (0xC0-0xFF)
	// Uppercase A variants
	latin1SupplementMap[0xC0] = 'A' // À
	latin1SupplementMap[0xC1] = 'A' // Á
	latin1SupplementMap[0xC2] = 'A' // Â
	latin1SupplementMap[0xC3] = 'A' // Ã
	latin1SupplementMap[0xC4] = 'A' // Ä
	latin1SupplementMap[0xC5] = 'A' // Å
	latin1SupplementMap[0xC6] = 'A' // Æ -> A (could be "AE" but single char is simpler)

	// Uppercase C
	latin1SupplementMap[0xC7] = 'C' // Ç

	// Uppercase E variants
	latin1SupplementMap[0xC8] = 'E' // È
	latin1SupplementMap[0xC9] = 'E' // É
	latin1SupplementMap[0xCA] = 'E' // Ê
	latin1SupplementMap[0xCB] = 'E' // Ë

	// Uppercase I variants
	latin1SupplementMap[0xCC] = 'I' // Ì
	latin1SupplementMap[0xCD] = 'I' // Í
	latin1SupplementMap[0xCE] = 'I' // Î
	latin1SupplementMap[0xCF] = 'I' // Ï

	// Uppercase D, N
	latin1SupplementMap[0xD0] = 'D' // Ð
	latin1SupplementMap[0xD1] = 'N' // Ñ

	// Uppercase O variants
	latin1SupplementMap[0xD2] = 'O' // Ò
	latin1SupplementMap[0xD3] = 'O' // Ó
	latin1SupplementMap[0xD4] = 'O' // Ô
	latin1SupplementMap[0xD5] = 'O' // Õ
	latin1SupplementMap[0xD6] = 'O' // Ö
	latin1SupplementMap[0xD8] = 'O' // Ø

	// Uppercase U variants
	latin1SupplementMap[0xD9] = 'U' // Ù
	latin1SupplementMap[0xDA] = 'U' // Ú
	latin1SupplementMap[0xDB] = 'U' // Û
	latin1SupplementMap[0xDC] = 'U' // Ü

	// Uppercase Y
	latin1SupplementMap[0xDD] = 'Y' // Ý
	latin1SupplementMap[0xDE] = 'T' // Þ (Thorn)

	// Lowercase sharp s
	latin1SupplementMap[0xDF] = 's' // ß

	// Lowercase a variants
	latin1SupplementMap[0xE0] = 'a' // à
	latin1SupplementMap[0xE1] = 'a' // á
	latin1SupplementMap[0xE2] = 'a' // â
	latin1SupplementMap[0xE3] = 'a' // ã
	latin1SupplementMap[0xE4] = 'a' // ä
	latin1SupplementMap[0xE5] = 'a' // å
	latin1SupplementMap[0xE6] = 'a' // æ -> a (could be "ae" but single char is simpler)

	// Lowercase c
	latin1SupplementMap[0xE7] = 'c' // ç

	// Lowercase e variants
	latin1SupplementMap[0xE8] = 'e' // è
	latin1SupplementMap[0xE9] = 'e' // é
	latin1SupplementMap[0xEA] = 'e' // ê
	latin1SupplementMap[0xEB] = 'e' // ë

	// Lowercase i variants
	latin1SupplementMap[0xEC] = 'i' // ì
	latin1SupplementMap[0xED] = 'i' // í
	latin1SupplementMap[0xEE] = 'i' // î
	latin1SupplementMap[0xEF] = 'i' // ï

	// Lowercase d, n
	latin1SupplementMap[0xF0] = 'd' // ð
	latin1SupplementMap[0xF1] = 'n' // ñ

	// Lowercase o variants
	latin1SupplementMap[0xF2] = 'o' // ò
	latin1SupplementMap[0xF3] = 'o' // ó
	latin1SupplementMap[0xF4] = 'o' // ô
	latin1SupplementMap[0xF5] = 'o' // õ
	latin1SupplementMap[0xF6] = 'o' // ö
	latin1SupplementMap[0xF8] = 'o' // ø

	// Lowercase u variants
	latin1SupplementMap[0xF9] = 'u' // ù
	latin1SupplementMap[0xFA] = 'u' // ú
	latin1SupplementMap[0xFB] = 'u' // û
	latin1SupplementMap[0xFC] = 'u' // ü

	// Lowercase y
	latin1SupplementMap[0xFD] = 'y' // ý
	latin1SupplementMap[0xFE] = 't' // þ (thorn)
	latin1SupplementMap[0xFF] = 'y' // ÿ

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

// isTrim returns true if the byte is to be trimmed at the ends.
func isTrim(b byte) bool { return b == '.' || b == '_' || b == '-' }

// appendSanitizedMetricName converts *any* string into something that StatsD / Graphite
// accepts without complaints.
func appendSanitizedMetricName(dst []byte, raw string) []byte {
	nameLen := 0
	orig := len(dst)
	if raw == "" {
		if len(dst) == 0 {
			return append(dst, "_unnamed_"...)
		}
		return dst
	}
	// ── 1. accent folding (creates one temporary ↴)
	// tmp := stripUnicodeAccents([]byte(raw))

	// ── 2. run the same ASCII sanitizer, but write into dst
	lastWasRepl := false
	for i := 0; i < len(raw); i++ {
		c := byte(raw[i])

		if c < 128 && valid[c] {
			// ASCII valid chars
			dst = append(dst, c)
			nameLen++
			lastWasRepl = false
		} else if c >= 0xC2 && c <= 0xC3 && i+1 < len(raw) {
			// Check for 2-byte UTF-8 sequences that are common accented letters
			c2 := byte(raw[i+1])
			if c2 >= 0x80 && c2 <= 0xBF { // Valid second byte
				// Decode the 2-byte sequence
				codepoint := uint16(c&0x1F)<<6 | uint16(c2&0x3F)

				// Map common accented characters (U+00C0-U+00FF range)
				if codepoint >= 0xC0 && codepoint <= 0xFF {
					mapped := latin1SupplementMap[codepoint]
					if valid[mapped] {
						dst = append(dst, mapped)
						nameLen++
						lastWasRepl = false
						i++ // Skip the second byte
						continue
					}
				}
			}
			// If we get here, treat as invalid
			if !lastWasRepl {
				dst = append(dst, replacement)
				nameLen++
				lastWasRepl = true
			}
		} else {
			// Everything else (3-byte, 4-byte sequences, invalid chars)
			// Skip continuation bytes (0x80-0xBF) to avoid creating invalid UTF-8
			for i+1 < len(raw) && (raw[i+1]&0xC0) == 0x80 {
				i++
			}
			if !lastWasRepl {
				dst = append(dst, replacement)
				nameLen++
				lastWasRepl = true
			}
		}

		if nameLen >= maxLen {
			break
		}
	}

	// 3. trim leading / trailing '.', '_' or '-'
	start, end := orig, len(dst)
	for start < end && isTrim(dst[start]) {
		start++
	}
	for end > start && isTrim(dst[end-1]) {
		end--
	}

	// 4. compact if we trimmed something
	if start > orig || end < len(dst) {
		copy(dst[orig:], dst[start:end])
		dst = dst[:orig+(end-start)]
	}

	// 5. fallback if everything vanished
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
