// +build windows

package procstats

import "github.com/segmentio/stats"

type RusageStats struct {
	// TODO
}

func NewRusageStats(client stats.Client) *RusageStats {
	return &RusageStats{
	// TODO
	}
}

func (ru *RusageStats) Collect() {
	// TODO
}
