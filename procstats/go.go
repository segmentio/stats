package procstats

import (
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/segmentio/stats"
)

func init() {
	stats.DefaultEngine.SetHistogramBucket("go.memstats.gc_pause.seconds",
		1e-6, // 1us
		1e-5, // 10us
		1e-4, // 100us
		1e-3, // 1ms
		1e-2, // 10ms
		1e-1, // 100ms
		1,    // 1s
		math.Inf(+1),
	)
}

// GoMetrics is a metric collector that reports metrics from the Go runtime.
type GoMetrics struct {
	// Runtime info.
	numCPU       stats.Gauge
	numGoroutine stats.Gauge
	numCgoCall   stats.Counter

	// General statistics.
	alloc      stats.Gauge   // bytes allocated (even if freed)
	totalAlloc stats.Counter // bytes allocated (even if freed)
	lookups    stats.Counter // number of pointer lookups
	mallocs    stats.Counter // number of mallocs
	frees      stats.Counter // number of frees

	// Main allocation heap statistics.
	heapAlloc    stats.Gauge   // bytes allocated and not yet freed
	heapSys      stats.Gauge   // bytes obtained from system
	heapIdle     stats.Gauge   // bytes in idle spans
	heapInuse    stats.Gauge   // bytes in non-idle span
	heapReleased stats.Counter // bytes released to the OS
	heapObjects  stats.Gauge   // total number of allocated objects

	// Low-level fixed-size structure allocator statistics.
	stackInuse  stats.Gauge // bytes used by stack allocator
	stackSys    stats.Gauge
	mSpanInuse  stats.Gauge // mspan structures
	mSpanSys    stats.Gauge
	mCacheInuse stats.Gauge // mcache structures
	mCacheSys   stats.Gauge
	buckHashSys stats.Gauge // profiling bucket hash table
	gcSys       stats.Gauge // GC metadata
	otherSys    stats.Gauge // other system allocations

	// Garbage collector statistics.
	numGC         stats.Counter // number of garbage collections
	nextGC        stats.Gauge   // next collection will happen when HeapAlloc ≥ this amount
	lastGC        stats.Gauge   // end time of last collection (nanoseconds since 1970)
	pauses        stats.Histogram
	gcCPUFraction stats.Gauge // fraction of CPU time used by GC

	// cache
	ms        runtime.MemStats
	gc        debug.GCStats
	lastNumGC uint64
}

// NewGoMetrics creates a new collector for the Go runtime that produces metrics
// on the default stats engine.
func NewGoMetrics() *GoMetrics {
	return NewGoMetricsWith(stats.DefaultEngine)
}

// NewGoMetricsWith creates a new collector for the Go unrtime that producers
// metrics on eng.
func NewGoMetricsWith(eng *stats.Engine) *GoMetrics {
	tags := append(make([]stats.Tag, 0, 3),
		stats.Tag{"runtime", "go"},
		stats.Tag{"version", runtime.Version()},
	)

	g := &GoMetrics{
		numCPU:       *eng.Gauge("go.runtime.cpu.num", tags...),
		numGoroutine: *eng.Gauge("go.runtime.goroutine.num", tags...),
		numCgoCall:   *eng.Counter("go.runtime.cgo.calls", tags...),
	}

	tagsTotal := append(tags, stats.Tag{"type", "total"})
	g.alloc = *eng.Gauge("go.memstats.alloc.bytes", tagsTotal...)
	g.totalAlloc = *eng.Counter("go.memstats.total_alloc.bytes", tagsTotal...)
	g.lookups = *eng.Counter("go.memstats.lookups.count", tagsTotal...)
	g.mallocs = *eng.Counter("go.memstats.mallocs.count", tagsTotal...)
	g.frees = *eng.Counter("go.memstats.frees.count", tagsTotal...)

	tagsHeap := append(tags, stats.Tag{"type", "heap"})
	g.heapAlloc = *eng.Gauge("go.memstats.alloc.bytes", tagsHeap...)
	g.heapSys = *eng.Gauge("go.memstats.sys.bytes", tagsHeap...)
	g.heapIdle = *eng.Gauge("go.memstats.idle.bytes", tagsHeap...)
	g.heapInuse = *eng.Gauge("go.memstats.inuse.bytes", tagsHeap...)
	g.heapReleased = *eng.Counter("go.memstats.released.bytes", tagsHeap...)
	g.heapObjects = *eng.Gauge("go.memstats.objects.count", tagsHeap...)

	tagsStack := append(tags, stats.Tag{"type", "stack"})
	g.stackInuse = *eng.Gauge("go.memstats.inuse.bytes", tagsStack...)
	g.stackSys = *eng.Gauge("go.memstats.sys.bytes", tagsStack...)

	tagsMSpan := append(tags, stats.Tag{"type", "mspan"})
	g.mSpanInuse = *eng.Gauge("go.memstats.inuse.bytes", tagsMSpan...)
	g.mSpanSys = *eng.Gauge("go.memstats.sys.bytes", tagsMSpan...)

	tagsMCache := append(tags, stats.Tag{"type", "mcache"})
	g.mCacheInuse = *eng.Gauge("go.memstats.inuse.bytes", tagsMCache...)
	g.mCacheSys = *eng.Gauge("go.memstats.sys.bytes", tagsMCache...)

	tagsBuckHash := append(tags, stats.Tag{"type", "bucket_hash_table"})
	g.buckHashSys = *eng.Gauge("go.memstats.sys.bytes", tagsBuckHash...)

	tagsGC := append(tags, stats.Tag{"type", "gc"})
	g.gcSys = *eng.Gauge("go.memstats.sys.bytes", tagsGC...)

	otherTags := append(tags, stats.Tag{"type", "other"})
	g.otherSys = *eng.Gauge("go.memstats.sys.bytes", otherTags...)

	g.numGC = *eng.Counter("go.memstats.gc.count", tags...)
	g.nextGC = *eng.Gauge("go.memstats.gc_next.bytes", tags...)
	g.lastGC = *eng.Gauge("go.memstats.gc_last.seconds", tags...)
	g.pauses = *eng.Histogram("go.memstats.gc_pause.seconds", tags...)
	g.gcCPUFraction = *eng.Gauge("go.memstats.gc_cpu.fraction", tags...)

	g.lastNumGC = uint64(g.gc.NumGC)
	return g
}

// Collect satisfies the Collector interface.
func (g *GoMetrics) Collect() {
	numCgoCall := uint64(runtime.NumCgoCall())
	g.numCPU.Set(float64(runtime.NumCPU()))
	g.numGoroutine.Set(float64(runtime.NumGoroutine()))
	g.numCgoCall.Set(float64(numCgoCall))

	collectMemoryStats(&g.ms, &g.gc, g.lastNumGC)
	g.updateMemStats(time.Now())
}

func (g *GoMetrics) updateMemStats(now time.Time) {
	g.alloc.Set(float64(g.ms.Alloc))
	g.totalAlloc.Set(float64(g.ms.TotalAlloc))
	g.lookups.Set(float64(g.ms.Lookups))
	g.mallocs.Set(float64(g.ms.Mallocs))
	g.frees.Set(float64(g.ms.Frees))

	g.heapAlloc.Set(float64(g.ms.HeapAlloc))
	g.heapSys.Set(float64(g.ms.HeapSys))
	g.heapIdle.Set(float64(g.ms.HeapIdle))
	g.heapInuse.Set(float64(g.ms.HeapInuse))
	g.heapReleased.Set(float64(g.ms.HeapReleased))
	g.heapObjects.Set(float64(g.ms.HeapObjects))

	g.stackInuse.Set(float64(g.ms.StackInuse))
	g.stackSys.Set(float64(g.ms.StackSys))
	g.mSpanInuse.Set(float64(g.ms.MSpanInuse))
	g.mSpanSys.Set(float64(g.ms.MSpanSys))
	g.mCacheInuse.Set(float64(g.ms.MCacheInuse))
	g.mCacheSys.Set(float64(g.ms.MCacheSys))
	g.buckHashSys.Set(float64(g.ms.BuckHashSys))
	g.gcSys.Set(float64(g.ms.GCSys))
	g.otherSys.Set(float64(g.ms.OtherSys))

	g.numGC.Set(float64(g.gc.NumGC))
	g.nextGC.Set(float64(g.ms.NextGC))
	g.lastGC.Set(now.Sub(g.gc.LastGC).Seconds())
	g.gcCPUFraction.Set(float64(g.ms.GCCPUFraction))

	for _, pause := range g.gc.Pause {
		g.pauses.Observe(pause.Seconds())
		// TODO: support reporting metrics at a specific time (in the past), not
		// all collection systems support it (datadog doesn't for example) but it
		// could help get a more accurate view on those that do.
		//
		// g.pauses.ObserveAt(pause.Seconds(), g.gc.PauseEnd[i])
	}
}

func collectMemoryStats(ms *runtime.MemStats, gc *debug.GCStats, lastNumGC uint64) {
	collectMemStats(ms)
	collectGCStats(gc, lastNumGC)
}

func collectMemStats(ms *runtime.MemStats) {
	runtime.ReadMemStats(ms)
}

func collectGCStats(gc *debug.GCStats, lastNumGC uint64) {
	debug.ReadGCStats(gc)
	stripOutdatedGCPauses(gc, lastNumGC)
}

func stripOutdatedGCPauses(gc *debug.GCStats, lastNumGC uint64) {
	for i := range gc.Pause {
		if num := uint64(gc.NumGC) - uint64(i); num <= lastNumGC {
			gc.Pause = gc.Pause[:i]
			gc.PauseEnd = gc.PauseEnd[:i]
			break
		}
	}
}
