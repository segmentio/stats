package datadog

import (
	"math"
)

// Datagram format: https://docs.datadoghq.com/developers/dogstatsd/datagram_shell

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
