package procstats

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/segmentio/stats"
)

type GoMetrics struct {
	// Runtime info.
	NumCPU       stats.Gauge
	NumGoroutine stats.Gauge
	NumCgoCall   stats.Counter

	// General statistics.
	Alloc      stats.Gauge     // bytes allocated (even if freed)
	TotalAlloc stats.Counter   // bytes allocated (even if freed)
	Lookups    stats.Counter   // number of pointer lookups
	Mallocs    stats.Counter   // number of mallocs
	Frees      stats.Counter   // number of frees
	Collects   stats.Histogram // time spent collecting memory stats

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

	// Per-size allocation statistics.
	// 61 is NumSizeClasses in the C code.Collector
	bySize [len(((*runtime.MemStats)(nil)).BySize)]struct {
		tags stats.Tags

		// Last observed values, used to compute the difference and know how
		// much should be added to the counters.
		lastMallocs uint64
		lastFrees   uint64
	}

	// Last observed values for high-level counters.
	lastNumCgoCall   uint64
	lastTotalAlloc   uint64
	lastHeapReleased uint64
	lastLookups      uint64
	lastMallocs      uint64
	lastFrees        uint64
	lastNumGC        uint64

	// cache
	ms runtime.MemStats
	gc debug.GCStats
}

func NewGoMetrics(client stats.Client, tags ...stats.Tag) *GoMetrics {
	tags = append(tags,
		stats.Tag{"runtime", "go"},
		stats.Tag{"version", runtime.Version()},
	)

	g := &GoMetrics{
		NumCPU:       client.Gauge("go.runtime.cpu.num", tags...),
		NumGoroutine: client.Gauge("go.runtime.goroutine.num", tags...),
		NumCgoCall:   client.Counter("go.runtime.cgo.calls", tags...),
	}

	tagsTotal := append(tags, stats.Tag{"type", "total"})
	g.Alloc = client.Gauge("go.memstats.alloc.bytes", tagsTotal...)
	g.TotalAlloc = client.Counter("go.memstats.total_alloc.bytes", tagsTotal...)
	g.Lookups = client.Counter("go.memstats.lookups.count", tagsTotal...)
	g.Mallocs = client.Counter("go.memstats.mallocs.count", tagsTotal...)
	g.Frees = client.Counter("go.memstats.frees.count", tagsTotal...)
	g.Collects = client.Histogram("go.memstats.collects.seconds", tagsTotal...)

	tagsHeap := append(tags, stats.Tag{"type", "heap"})
	g.HeapAlloc = client.Gauge("go.memstats.alloc.bytes", tagsHeap...)
	g.HeapSys = client.Gauge("go.memstats.sys.bytes", tagsHeap...)
	g.HeapIdle = client.Gauge("go.memstats.idle.bytes", tagsHeap...)
	g.HeapInuse = client.Gauge("go.memstats.inuse.bytes", tagsHeap...)
	g.HeapReleased = client.Counter("go.memstats.released.bytes", tagsHeap...)
	g.HeapObjects = client.Gauge("go.memstats.objects.count", tagsHeap...)

	tagsStack := append(tags, stats.Tag{"type", "stack"})
	g.StackInuse = client.Gauge("go.memstats.inuse.bytes", tagsStack...)
	g.StackSys = client.Gauge("go.memstats.sys.bytes", tagsStack...)

	tagsMSpan := append(tags, stats.Tag{"type", "mspan"})
	g.MSpanInuse = client.Gauge("go.memstats.inuse.bytes", tagsMSpan...)
	g.MSpanSys = client.Gauge("go.memstats.sys.bytes", tagsMSpan...)

	tagsMCache := append(tags, stats.Tag{"type", "mcache"})
	g.MCacheInuse = client.Gauge("go.memstats.inuse.bytes", tagsMCache...)
	g.MCacheSys = client.Gauge("go.memstats.sys.bytes", tagsMCache...)

	tagsBuckHash := append(tags, stats.Tag{"type", "bucket_hash_table"})
	g.BuckHashSys = client.Gauge("go.memstats.sys.bytes", tagsBuckHash...)

	tagsGC := append(tags, stats.Tag{"type", "gc"})
	g.GCSys = client.Gauge("go.memstats.sys.bytes", tagsGC...)

	otherTags := append(tags, stats.Tag{"type", "other"})
	g.OtherSys = client.Gauge("go.memstats.sys.bytes", otherTags...)

	g.NumGC = client.Counter("go.memstats.gc.count", tags...)
	g.NextGC = client.Gauge("go.memstats.gc_next.bytes", tags...)
	g.LastGC = client.Gauge("go.memstats.gc_last.seconds", tags...)
	g.Pauses = client.Histogram("go.memstats.gc_pause.seconds", tags...)
	g.GCCPUFraction = client.Gauge("go.memstats.gc_cpu.fraction", tags...)

	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)

	for i := 0; i != len(g.bySize); i++ {
		g.bySize[i].tags = stats.Tags{
			{"type", "bucket"},
			{"size", strconv.FormatUint(uint64(ms.BySize[i].Size), 10)},
		}
	}

	return g
}

func (g *GoMetrics) Collect() {
	numCgoCall := uint64(runtime.NumCgoCall())
	g.NumCPU.Set(float64(runtime.NumCPU()))
	g.NumGoroutine.Set(float64(runtime.NumGoroutine()))
	g.NumCgoCall.Add(float64(numCgoCall - g.lastNumCgoCall))
	g.lastNumCgoCall = numCgoCall

	msd, gcd := collectMemoryStats(&g.ms, &g.gc, g.lastNumGC)
	g.updateMemStats(time.Now(), msd, gcd)
}

func (g *GoMetrics) updateMemStats(now time.Time, msd time.Duration, gcd time.Duration) {
	ms := &g.ms
	gc := &g.gc

	g.Alloc.Set(float64(ms.Alloc))
	g.TotalAlloc.Add(float64(ms.TotalAlloc - g.lastTotalAlloc))
	g.Lookups.Add(float64(ms.Lookups - g.lastLookups))
	g.Mallocs.Add(float64(ms.Mallocs - g.lastMallocs))
	g.Frees.Add(float64(ms.Frees - g.lastFrees))
	g.Collects.Observe(msd.Seconds(), stats.Tag{"step", "memstats"})
	g.Collects.Observe(gcd.Seconds(), stats.Tag{"step", "gcstats"})

	g.HeapAlloc.Set(float64(ms.HeapAlloc))
	g.HeapSys.Set(float64(ms.HeapSys))
	g.HeapIdle.Set(float64(ms.HeapIdle))
	g.HeapInuse.Set(float64(ms.HeapInuse))
	g.HeapReleased.Add(float64(ms.HeapReleased - g.lastHeapReleased))
	g.HeapObjects.Set(float64(ms.HeapObjects))

	g.StackInuse.Set(float64(ms.StackInuse))
	g.StackSys.Set(float64(ms.StackSys))
	g.MSpanInuse.Set(float64(ms.MSpanInuse))
	g.MSpanSys.Set(float64(ms.MSpanSys))
	g.MCacheInuse.Set(float64(ms.MCacheInuse))
	g.MCacheSys.Set(float64(ms.MCacheSys))
	g.BuckHashSys.Set(float64(ms.BuckHashSys))
	g.GCSys.Set(float64(ms.GCSys))
	g.OtherSys.Set(float64(ms.OtherSys))

	g.NumGC.Add(float64(uint64(gc.NumGC) - g.lastNumGC))
	g.NextGC.Set(float64(ms.NextGC))
	g.LastGC.Set(now.Sub(gc.LastGC).Seconds())
	g.GCCPUFraction.Set(float64(ms.GCCPUFraction))

	for i := 0; i != len(g.bySize); i++ {
		a := &(ms.BySize[i])
		b := &(g.bySize[i])

		g.Mallocs.Add(float64(a.Mallocs-b.lastMallocs), b.tags...)
		g.Frees.Add(float64(a.Frees-b.lastFrees), b.tags...)

		b.lastMallocs = a.Mallocs
		b.lastFrees = a.Frees
	}

	for i, pause := range gc.Pause {
		g.Pauses.ObserveAt(pause.Seconds(), gc.PauseEnd[i])
	}

	g.lastTotalAlloc = ms.TotalAlloc
	g.lastHeapReleased = ms.HeapReleased
	g.lastLookups = ms.Lookups
	g.lastMallocs = ms.Mallocs
	g.lastFrees = ms.Frees
	g.lastNumGC = uint64(gc.NumGC)
}

func collectMemoryStats(ms *runtime.MemStats, gc *debug.GCStats, lastNumGC uint64) (time.Duration, time.Duration) {
	return collectMemStats(ms), collectGCStats(gc, lastNumGC)
}

func collectMemStats(ms *runtime.MemStats) time.Duration {
	t := time.Now()
	runtime.ReadMemStats(ms)
	return time.Now().Sub(t)
}

func collectGCStats(gc *debug.GCStats, lastNumGC uint64) time.Duration {
	t := time.Now()
	debug.ReadGCStats(gc)
	d := time.Now().Sub(t)
	stripOutdatedGCPauses(gc, lastNumGC)
	return d
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
