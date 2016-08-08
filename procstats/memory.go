package procstats

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/segmentio/stats"
)

type MemoryStats struct {
	// General statistics.
	Alloc    stats.Counter   // bytes allocated (even if freed)
	Release  stats.Counter   // bytes released to the OS
	Lookups  stats.Counter   // number of pointer lookups
	Mallocs  stats.Counter   // number of mallocs
	Frees    stats.Counter   // number of frees
	Collects stats.Histogram // time spent collecting memory stats

	// Main allocation heap statistics.
	HeapActive  stats.Gauge // bytes allocated and not yet freed
	HeapSystem  stats.Gauge // bytes obtained from system
	HeapIdle    stats.Gauge // bytes in idle spans
	HeapUsed    stats.Gauge // bytes in non-idle span
	HeapObjects stats.Gauge // total number of allocated objects

	// Low-level fixed-size structure allocator statistics.
	StackUsed      stats.Gauge // bytes used by stack allocator
	StackSystem    stats.Gauge
	MSpanUsed      stats.Gauge // mspan structures
	MSpanSystem    stats.Gauge
	MCacheUsed     stats.Gauge // mcache structures
	MCacheSystem   stats.Gauge
	BuckHashSystem stats.Gauge // profiling bucket hash table
	GCSystem       stats.Gauge // GC metadata
	OtherSystem    stats.Gauge // other system allocations

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
	lastAlloc   uint64
	lastRelease uint64
	lastLookups uint64
	lastMallocs uint64
	lastFrees   uint64
	lastNumGC   uint64

	// cache
	ms runtime.MemStats
	gc debug.GCStats
}

func NewMemoryStats(client stats.Client, tags ...stats.Tag) *MemoryStats {
	tags = append(tags, stats.Tag{"go_version", runtime.Version()})
	mem := &MemoryStats{}

	tagsTotal := append(tags, stats.Tag{"type", "total"})
	mem.Alloc = client.Counter("memory.allocated.bytes", tagsTotal...)
	mem.Release = client.Counter("memory.released.bytes", tagsTotal...)
	mem.Lookups = client.Counter("memory.lookups.count", tagsTotal...)
	mem.Mallocs = client.Counter("memory.malloc.count", tagsTotal...)
	mem.Frees = client.Counter("memory.free.count", tagsTotal...)
	mem.Collects = client.Histogram("memory.collect.seconds", tagsTotal...)

	tagsHeap := append(tags, stats.Tag{"type", "heap"})
	mem.HeapActive = client.Gauge("memory.active.bytes", tagsHeap...)
	mem.HeapSystem = client.Gauge("memory.system.bytes", tagsHeap...)
	mem.HeapIdle = client.Gauge("memory.idle.bytes", tagsHeap...)
	mem.HeapUsed = client.Gauge("memory.used.bytes", tagsHeap...)
	mem.HeapObjects = client.Gauge("memory.objects.count", tagsHeap...)

	tagsStack := append(tags, stats.Tag{"type", "stack"})
	mem.StackUsed = client.Gauge("memory.used.bytes", tagsStack...)
	mem.StackSystem = client.Gauge("memory.system.bytes", tagsStack...)

	tagsMSpan := append(tags, stats.Tag{"type", "mspan"})
	mem.MSpanUsed = client.Gauge("memory.used.bytes", tagsMSpan...)
	mem.MSpanSystem = client.Gauge("memory.system.bytes", tagsMSpan...)

	tagsMCache := append(tags, stats.Tag{"type", "mcache"})
	mem.MCacheUsed = client.Gauge("memory.used.bytes", tagsMCache...)
	mem.MCacheSystem = client.Gauge("memory.system.bytes", tagsMCache...)

	tagsBuckHash := append(tags, stats.Tag{"type", "bucket_hash_table"})
	mem.BuckHashSystem = client.Gauge("memory.system.bytes", tagsBuckHash...)

	tagsGC := append(tags, stats.Tag{"type", "gc"})
	mem.GCSystem = client.Gauge("memory.system.bytes", tagsGC...)

	otherTags := append(tags, stats.Tag{"type", "other"})
	mem.OtherSystem = client.Gauge("memory.system.bytes", otherTags...)

	mem.NumGC = client.Counter("memory.gc.count", tags...)
	mem.NextGC = client.Gauge("memory.gc_next.bytes", tags...)
	mem.LastGC = client.Gauge("memory.gc_last.seconds", tags...)
	mem.Pauses = client.Histogram("memory.gc_pause.seconds", tags...)
	mem.GCCPUFraction = client.Gauge("memory.gc_cpu.fraction", tags...)

	ms := &runtime.MemStats{}
	runtime.ReadMemStats(ms)

	for i := 0; i != len(mem.bySize); i++ {
		mem.bySize[i].tags = stats.Tags{
			{"type", "bucket"},
			{"size", strconv.FormatUint(uint64(ms.BySize[i].Size), 10)},
		}
	}

	return mem
}

func (mem *MemoryStats) Collect() {
	msd, gcd := collectMemoryStats(&mem.ms, &mem.gc, mem.lastNumGC)
	mem.report(time.Now(), msd, gcd)
}

func (mem *MemoryStats) report(now time.Time, msd time.Duration, gcd time.Duration) {
	ms := &mem.ms
	gc := &mem.gc

	mem.Alloc.Add(float64(ms.TotalAlloc - mem.lastAlloc))
	mem.Release.Add(float64(ms.HeapReleased - mem.lastRelease))
	mem.Lookups.Add(float64(ms.Lookups - mem.lastLookups))
	mem.Mallocs.Add(float64(ms.Mallocs - mem.lastMallocs))
	mem.Frees.Add(float64(ms.Frees - mem.lastFrees))
	mem.Collects.Observe(msd.Seconds(), stats.Tag{"step", "memstats"})
	mem.Collects.Observe(gcd.Seconds(), stats.Tag{"step", "gcstats"})

	mem.HeapActive.Set(float64(ms.HeapAlloc))
	mem.HeapSystem.Set(float64(ms.HeapSys))
	mem.HeapIdle.Set(float64(ms.HeapIdle))
	mem.HeapUsed.Set(float64(ms.HeapInuse))
	mem.HeapObjects.Set(float64(ms.HeapObjects))

	mem.StackUsed.Set(float64(ms.StackInuse))
	mem.StackSystem.Set(float64(ms.StackSys))
	mem.MSpanUsed.Set(float64(ms.MSpanInuse))
	mem.MSpanSystem.Set(float64(ms.MSpanSys))
	mem.MCacheUsed.Set(float64(ms.MCacheInuse))
	mem.MCacheSystem.Set(float64(ms.MCacheSys))
	mem.BuckHashSystem.Set(float64(ms.BuckHashSys))
	mem.GCSystem.Set(float64(ms.GCSys))
	mem.OtherSystem.Set(float64(ms.OtherSys))

	mem.NumGC.Add(float64(uint64(gc.NumGC) - mem.lastNumGC))
	mem.NextGC.Set(float64(ms.NextGC))
	mem.LastGC.Set(now.Sub(gc.LastGC).Seconds())
	mem.GCCPUFraction.Set(float64(ms.GCCPUFraction))

	for i := 0; i != len(mem.bySize); i++ {
		a := &(ms.BySize[i])
		b := &(mem.bySize[i])

		mem.Mallocs.Add(float64(a.Mallocs-b.lastMallocs), b.tags...)
		mem.Frees.Add(float64(a.Frees-b.lastFrees), b.tags...)

		b.lastMallocs = a.Mallocs
		b.lastFrees = a.Frees
	}

	for i, pause := range gc.Pause {
		mem.Pauses.ObserveAt(pause.Seconds(), gc.PauseEnd[i])
	}

	mem.lastAlloc = ms.TotalAlloc
	mem.lastRelease = ms.HeapReleased
	mem.lastLookups = ms.Lookups
	mem.lastMallocs = ms.Mallocs
	mem.lastFrees = ms.Frees
	mem.lastNumGC = uint64(gc.NumGC)
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
