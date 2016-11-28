package procstats

import (
	"runtime"
	"runtime/debug"
	"time"

	"github.com/segmentio/stats"
)

type GoMetrics struct {
	// Runtime info.
	NumCPU       stats.Gauge
	NumGoroutine stats.Gauge
	NumCgoCall   stats.Counter

	// General statistics.
	Alloc      stats.Gauge   // bytes allocated (even if freed)
	TotalAlloc stats.Counter // bytes allocated (even if freed)
	Lookups    stats.Counter // number of pointer lookups
	Mallocs    stats.Counter // number of mallocs
	Frees      stats.Counter // number of frees

	// Main allocation heap statistics.
	HeapAlloc    stats.Gauge   // bytes allocated and not yet freed
	HeapSys      stats.Gauge   // bytes obtained from system
	HeapIdle     stats.Gauge   // bytes in idle spans
	HeapInuse    stats.Gauge   // bytes in non-idle span
	HeapReleased stats.Counter // bytes released to the OS
	HeapObjects  stats.Gauge   // total number of allocated objects

	// Low-level fixed-size structure allocator statistics.
	StackInuse  stats.Gauge // bytes used by stack allocator
	StackSys    stats.Gauge
	MSpanInuse  stats.Gauge // mspan structures
	MSpanSys    stats.Gauge
	MCacheInuse stats.Gauge // mcache structures
	MCacheSys   stats.Gauge
	BuckHashSys stats.Gauge // profiling bucket hash table
	GCSys       stats.Gauge // GC metadata
	OtherSys    stats.Gauge // other system allocations

	// Garbage collector statistics.
	NumGC         stats.Counter // number of garbage collections
	NextGC        stats.Gauge   // next collection will happen when HeapAlloc ≥ this amount
	LastGC        stats.Gauge   // end time of last collection (nanoseconds since 1970)
	Pauses        stats.Histogram
	GCCPUFraction stats.Gauge // fraction of CPU time used by GC

	// cache
	ms        runtime.MemStats
	gc        debug.GCStats
	lastNumGC uint64
}

func NewGoMetrics(eng *stats.Engine, tags ...stats.Tag) *GoMetrics {
	tags = append(tags,
		stats.Tag{"runtime", "go"},
		stats.Tag{"version", runtime.Version()},
	)

	g := &GoMetrics{
		NumCPU:       stats.MakeGauge(eng, "go.runtime.cpu.num", tags...),
		NumGoroutine: stats.MakeGauge(eng, "go.runtime.goroutine.num", tags...),
		NumCgoCall:   stats.MakeCounter(eng, "go.runtime.cgo.calls", tags...),
	}

	tagsTotal := append(tags, stats.Tag{"type", "total"})
	g.Alloc = stats.MakeGauge(eng, "go.memstats.alloc.bytes", tagsTotal...)
	g.TotalAlloc = stats.MakeCounter(eng, "go.memstats.total_alloc.bytes", tagsTotal...)
	g.Lookups = stats.MakeCounter(eng, "go.memstats.lookups.count", tagsTotal...)
	g.Mallocs = stats.MakeCounter(eng, "go.memstats.mallocs.count", tagsTotal...)
	g.Frees = stats.MakeCounter(eng, "go.memstats.frees.count", tagsTotal...)

	tagsHeap := append(tags, stats.Tag{"type", "heap"})
	g.HeapAlloc = stats.MakeGauge(eng, "go.memstats.alloc.bytes", tagsHeap...)
	g.HeapSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsHeap...)
	g.HeapIdle = stats.MakeGauge(eng, "go.memstats.idle.bytes", tagsHeap...)
	g.HeapInuse = stats.MakeGauge(eng, "go.memstats.inuse.bytes", tagsHeap...)
	g.HeapReleased = stats.MakeCounter(eng, "go.memstats.released.bytes", tagsHeap...)
	g.HeapObjects = stats.MakeGauge(eng, "go.memstats.objects.count", tagsHeap...)

	tagsStack := append(tags, stats.Tag{"type", "stack"})
	g.StackInuse = stats.MakeGauge(eng, "go.memstats.inuse.bytes", tagsStack...)
	g.StackSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsStack...)

	tagsMSpan := append(tags, stats.Tag{"type", "mspan"})
	g.MSpanInuse = stats.MakeGauge(eng, "go.memstats.inuse.bytes", tagsMSpan...)
	g.MSpanSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsMSpan...)

	tagsMCache := append(tags, stats.Tag{"type", "mcache"})
	g.MCacheInuse = stats.MakeGauge(eng, "go.memstats.inuse.bytes", tagsMCache...)
	g.MCacheSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsMCache...)

	tagsBuckHash := append(tags, stats.Tag{"type", "bucket_hash_table"})
	g.BuckHashSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsBuckHash...)

	tagsGC := append(tags, stats.Tag{"type", "gc"})
	g.GCSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", tagsGC...)

	otherTags := append(tags, stats.Tag{"type", "other"})
	g.OtherSys = stats.MakeGauge(eng, "go.memstats.sys.bytes", otherTags...)

	g.NumGC = stats.MakeCounter(eng, "go.memstats.gc.count", tags...)
	g.NextGC = stats.MakeGauge(eng, "go.memstats.gc_next.bytes", tags...)
	g.LastGC = stats.MakeGauge(eng, "go.memstats.gc_last.seconds", tags...)
	g.Pauses = stats.MakeHistogram(eng, "go.memstats.gc_pause.seconds", tags...)
	g.GCCPUFraction = stats.MakeGauge(eng, "go.memstats.gc_cpu.fraction", tags...)

	g.lastNumGC = uint64(g.gc.NumGC)
	return g
}

func (g *GoMetrics) Collect() {
	numCgoCall := uint64(runtime.NumCgoCall())
	g.NumCPU.Set(float64(runtime.NumCPU()))
	g.NumGoroutine.Set(float64(runtime.NumGoroutine()))
	g.NumCgoCall.Set(float64(numCgoCall))

	collectMemoryStats(&g.ms, &g.gc, g.lastNumGC)
	g.updateMemStats(time.Now())
}

func (g *GoMetrics) updateMemStats(now time.Time) {
	g.Alloc.Set(float64(g.ms.Alloc))
	g.TotalAlloc.Set(float64(g.ms.TotalAlloc))
	g.Lookups.Set(float64(g.ms.Lookups))
	g.Mallocs.Set(float64(g.ms.Mallocs))
	g.Frees.Set(float64(g.ms.Frees))

	g.HeapAlloc.Set(float64(g.ms.HeapAlloc))
	g.HeapSys.Set(float64(g.ms.HeapSys))
	g.HeapIdle.Set(float64(g.ms.HeapIdle))
	g.HeapInuse.Set(float64(g.ms.HeapInuse))
	g.HeapReleased.Set(float64(g.ms.HeapReleased))
	g.HeapObjects.Set(float64(g.ms.HeapObjects))

	g.StackInuse.Set(float64(g.ms.StackInuse))
	g.StackSys.Set(float64(g.ms.StackSys))
	g.MSpanInuse.Set(float64(g.ms.MSpanInuse))
	g.MSpanSys.Set(float64(g.ms.MSpanSys))
	g.MCacheInuse.Set(float64(g.ms.MCacheInuse))
	g.MCacheSys.Set(float64(g.ms.MCacheSys))
	g.BuckHashSys.Set(float64(g.ms.BuckHashSys))
	g.GCSys.Set(float64(g.ms.GCSys))
	g.OtherSys.Set(float64(g.ms.OtherSys))

	g.NumGC.Set(float64(g.gc.NumGC))
	g.NextGC.Set(float64(g.ms.NextGC))
	g.LastGC.Set(now.Sub(g.gc.LastGC).Seconds())
	g.GCCPUFraction.Set(float64(g.ms.GCCPUFraction))

	for _, pause := range g.gc.Pause {
		g.Pauses.Observe(pause.Seconds())
		// TODO: support reporting metrics at a specific time (in the past), not
		// all collection systems support it (datadog doesn't for example) but it
		// could help get a more accurate view on those that do.
		//
		// g.Pauses.ObserveAt(pause.Seconds(), g.gc.PauseEnd[i])
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
