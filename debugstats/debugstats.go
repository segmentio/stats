// Package debugstats simplifies metric troubleshooting by sending metrics to
// any io.Writer.
//
// By default, metrics will be printed to os.Stdout. Use the Dst and Grep fields
// to customize the output as appropriate.
package debugstats

import (
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/segmentio/stats/v5"
)

// Client will print out received metrics. If Dst is nil, metrics will be
// printed to stdout, otherwise they will be printed to Dst.
//
// You can optionally provide a Grep regexp to limit printed metrics to ones
// matching the regular expression.
type Client struct {
	Dst  io.Writer
	Grep *regexp.Regexp
}

func (c *Client) Write(p []byte) (int, error) {
	if c.Dst == nil {
		return os.Stdout.Write(p)
	}
	return c.Dst.Write(p)
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

func appendMeasure(b []byte, m stats.Measure) []byte {
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
			b = append(b, '|', 'd')
		}

		if n := len(m.Tags); n != 0 {
			b = append(b, '|', '#')

			for i, t := range m.Tags {
				if i != 0 {
					b = append(b, ',')
				}
				b = append(b, t.Name...)
				b = append(b, ':')
				b = append(b, t.Value...)
			}
		}

		b = append(b, '\n')
	}

	return b
}

func (c *Client) HandleMeasures(t time.Time, measures ...stats.Measure) {
	for i := range measures {
		m := &measures[i]

		// Process and output the measure
		out := make([]byte, 0)
		out = appendMeasure(out, *m)
		if c.Grep != nil && !c.Grep.Match(out) {
			continue // Skip this measure
		}

		fmt.Fprintf(c, "%s %s", t.Format(time.RFC3339), out)
	}
}
