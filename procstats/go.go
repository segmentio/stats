package procstats

import (
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/segmentio/stats"
)

func init() {
	stats.Buckets.Set("go.memstats:gc_pause.seconds",
		1*time.Microsecond,
		10*time.Microsecond,
		100*time.Microsecond,
		1*time.Millisecond,
		10*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
		math.Inf(+1),
	)
}

// GoMetrics is a metric collector that reports metrics from the Go runtime.
type GoMetrics struct {
	engine  *stats.Engine
	version string `tag:"version'`

	runtime struct {
		// Runtime info.
		numCPU       int `metric:"cpu.num"       type:"gauge"`
		numGoroutine int `metric:"goroutine.num" type:"gauge"`
		numCgoCall   int `metric:"cgo.calls"     type:"counter"`
	} `metric:"go.runtime"`

	memstats struct {
		// General statistics.
		total struct {
			alloc      uint64 `metric:"alloc.bytes"       type:"gauge"`   // bytes allocated (even if freed)
			totalAlloc uint64 `metric:"total_alloc.bytes" type:"counter"` // bytes allocated (even if freed)
			lookups    uint64 `metric:"lookups.count"     type:"counter"` // number of pointer lookups
			mallocs    uint64 `metric:"mallocs.count"     type:"counter"` // number of mallocs
			frees      uint64 `metric:"frees.count"       type:"counter"` // number of frees
			memtype    string `tag:"type"`
		}

		// Main allocation heap statistics.
		heap struct {
			alloc    uint64 `metric:"alloc.bytes"    type:"gauge"`   // bytes allocated and not yet freed
			sys      uint64 `metric:"sys.bytes"      type:"gauge"`   // bytes obtained from system
			idle     uint64 `metric:"idle.bytes"     type:"gauge"`   // bytes in idle spans
			inuse    uint64 `metric:"inuse.bytes"    type:"gauge"`   // bytes in non-idle span
			released uint64 `metric:"released.bytes" type:"counter"` // bytes released to the OS
			objects  uint64 `metric:"objects.count"  type:"gauge"`   // total number of allocated objects
			memtype  string `tag:"type"`
		}

		// Low-level fixed-size structure allocator statistics.
		stack struct {
			inuse   uint64 `metric:"" type:"gauge"` // bytes used by stack allocator
			sys     uint64 `metric:"" type:"gauge"`
			memtype string `tag:"type"`
		}

		mspan struct {
			inuse   uint64 `metric:"" type:"gauge"` // mspan structures
			sys     uint64 `metric:"" type:"gauge"`
			memtype string `tag:"type"`
		}

		mcache struct {
			inuse   uint64 `metric:"" type:"gauge"` // mcache structures
			sys     uint64 `metric:"" type:"gauge"`
			memtype string `tag:"type"`
		}

		buckhash struct {
			sys     uint64 `metric:"" type:"gauge"` // profiling bucket hash table
			memtype string `tag:"type"`
		}

		gc struct {
			sys     uint64 `metric:"sys.bytes" type:"gauge"` // GC metadata
			memtype string `tag:"type"`
		}

		other struct {
			sys     uint64 `metric:"sys.bytes" type:"gauge"` // other system allocations
			memtype string `tag:"type"`
		}

		// Garbage collector statistics.
		numGC         int64         `metric:"gc.count"         type:"counter"` // number of garbage collections
		nextGC        uint64        `metric:"gc_next.bytes"    type:"gauge"`   // next collection will happen when HeapAlloc ≥ this amount
		lastGC        time.Duration `metric:"gc_last.seconds"  type:"gauge"`   // end time of last collection (nanoseconds since 1970)
		pauses        time.Duration `metric:"gc_pause.seconds" type:"histogram"`
		gcCPUFraction float64       `metric:"gc_cpu.fraction"  type:"gauge"` // fraction of CPU time used by GC
	} `metric:"go.memstats"`

	// cache
	ms        runtime.MemStats
	gc        debug.GCStats
	lastNumGC int64
}

// NewGoMetrics creates a new collector for the Go runtime that produces metrics
// on the default stats engine.
func NewGoMetrics() *GoMetrics {
	return NewGoMetricsWith(stats.DefaultEngine)
}

// NewGoMetricsWith creates a new collector for the Go unrtime that producers
// metrics on eng.
func NewGoMetricsWith(eng *stats.Engine) *GoMetrics {
	g := &GoMetrics{
		engine:  eng,
		version: runtime.Version(),
	}

	g.memstats.total.memtype = "total"
	g.memstats.heap.memtype = "heap"
	g.memstats.stack.memtype = "stack"
	g.memstats.mspan.memtype = "mspan"
	g.memstats.mcache.memtype = "mcache"
	g.memstats.buckhash.memtype = "bucket_hash_table"
	g.memstats.gc.memtype = "gc"
	g.memstats.other.memtype = "other"

	debug.ReadGCStats(&g.gc)
	g.lastNumGC = g.gc.NumGC
	return g
}

// Collect satisfies the Collector interface.
func (g *GoMetrics) Collect() {
	now := time.Now()

	g.runtime.numCPU = runtime.NumCPU()
	g.runtime.numGoroutine = runtime.NumGoroutine()
	g.runtime.numCgoCall = int(runtime.NumCgoCall())

	collectMemoryStats(&g.ms, &g.gc, g.lastNumGC)
	g.updateMemStats(now)

	g.engine.ReportAt(now, g)
}

func (g *GoMetrics) updateMemStats(now time.Time) {
	g.memstats.total.alloc = g.ms.Alloc
	g.memstats.total.totalAlloc = g.ms.TotalAlloc
	g.memstats.total.lookups = g.ms.Lookups
	g.memstats.total.mallocs = g.ms.Mallocs
	g.memstats.total.frees = g.ms.Frees

	g.memstats.heap.alloc = g.ms.HeapAlloc
	g.memstats.heap.sys = g.ms.HeapSys
	g.memstats.heap.idle = g.ms.HeapIdle
	g.memstats.heap.inuse = g.ms.HeapInuse
	g.memstats.heap.released = g.ms.HeapReleased
	g.memstats.heap.objects = g.ms.HeapObjects

	g.memstats.stack.inuse = g.ms.StackInuse
	g.memstats.stack.sys = g.ms.StackSys
	g.memstats.mspan.inuse = g.ms.MSpanInuse
	g.memstats.mspan.sys = g.ms.MSpanSys
	g.memstats.mcache.inuse = g.ms.MCacheInuse
	g.memstats.mcache.sys = g.ms.MCacheSys
	g.memstats.buckhash.sys = g.ms.BuckHashSys
	g.memstats.gc.sys = g.ms.GCSys
	g.memstats.other.sys = g.ms.OtherSys

	g.memstats.numGC = g.gc.NumGC
	g.memstats.nextGC = g.ms.NextGC
	g.memstats.lastGC = now.Sub(g.gc.LastGC)
	g.memstats.gcCPUFraction = g.ms.GCCPUFraction

	for _, pause := range g.gc.Pause {
		g.memstats.pauses += pause
	}
}

func collectMemoryStats(ms *runtime.MemStats, gc *debug.GCStats, lastNumGC int64) {
	collectMemStats(ms)
	collectGCStats(gc, lastNumGC)
}

func collectMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
}

func collectGCStats(gc *debug.GCStats, lastNumGC int64) {
	debug.ReadGCStats(gc)
	stripOutdatedGCPauses(gc, lastNumGC)
}

func stripOutdatedGCPauses(gc *debug.GCStats, lastNumGC int64) {
	for i := range gc.Pause {
		if num := gc.NumGC - int64(i); num <= lastNumGC {
			gc.Pause = gc.Pause[:i]
			gc.PauseEnd = gc.PauseEnd[:i]
			break
		}
	}
}
