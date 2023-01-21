package datadog

import (
	"bytes"
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vertoforce/stats"
)

// Datagram format: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell

type serializer struct {
	conn             net.Conn
	bufferSize       int
	filters          map[string]struct{}
	distPrefixes     []string
	useDistributions bool
}

func (s *serializer) Write(b []byte) (int, error) {
	if s.conn == nil {
		return 0, io.ErrClosedPipe
	}

	if len(b) <= s.bufferSize {
		return s.conn.Write(b)
	}

	// When the serialized metrics are larger than the configured socket buffer
	// size we split them on '\n' characters.
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

// AppendMeasure is a formatting routine to append the dogstatsd protocol
// representation of a measure to a memory buffer.
// Tags listed in the s.filters are removed. (some tags may not be suitable for submission to DataDog)
// Histogram metrics will be sent as distribution type if the metric name matches s.distPrefixes
// DogStatsd Protocol Docs: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell?tab=metrics#the-dogstatsd-protocol
func (s *serializer) AppendMeasure(b []byte, m stats.Measure) []byte {
	for _, field := range m.Fields {
		b = append(b, m.Name...)
		if len(field.Name) != 0 {
			b = append(b, '.')
			b = append(b, field.Name...)
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

		if n := len(m.Tags); n != 0 {
			b = append(b, '|', '#')

			for i, t := range m.Tags {
				if _, ok := s.filters[t.Name]; !ok {
					if i != 0 {
						b = append(b, ',')
					}
					b = append(b, t.Name...)
					b = append(b, ':')
					b = append(b, t.Value...)
				}
			}
		}

		b = append(b, '\n')
	}

	return b
}

// sendDist determines whether to send a metric to datadog as histogram `h` type or
// distribution `d` type. It's a confusing setup because useDistributions and distPrefixes
// are independent implementations of a control mechanism for sending distributions that
// aren't elegantly coordinated
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
