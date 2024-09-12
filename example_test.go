package stats_test

import (
	"os"

	stats "github.com/segmentio/stats/v5"
	"github.com/segmentio/stats/v5/debugstats"
)

func Example() {
	handler := &debugstats.Client{Dst: os.Stdout}
	engine := stats.NewEngine("engine-name", handler)
	// Will print:
	//
	// 2024-12-18T14:53:57-08:00 engine-name.server.start:1|c
	//
	// to the console.
	engine.Incr("server.start")
	engine.Flush()
}
