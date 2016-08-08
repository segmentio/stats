// +build !windows

package procstats

import (
	"syscall"
	"time"

	"github.com/segmentio/stats"
)

type RusageStats struct {
	Utime    stats.Counter
	Stime    stats.Counter
	Maxrss   stats.Gauge
	Ixrss    stats.Gauge
	Idrss    stats.Gauge
	Isrss    stats.Gauge
	Minflt   stats.Counter
	Majflt   stats.Counter
	Nswap    stats.Counter
	Inblock  stats.Counter
	Oublock  stats.Counter
	Msgsnd   stats.Counter
	Msgrcv   stats.Counter
	Nsignals stats.Counter
	Nvcsw    stats.Counter
	Nivcsw   stats.Counter

	lastUtime    time.Duration
	lastStime    time.Duration
	lastMinflt   int64
	lastMajflt   int64
	lastNswap    int64
	lastInblock  int64
	lastOublock  int64
	lastMsgsnd   int64
	lastMsgrcv   int64
	lastNsignals int64
	lastNvcsw    int64
	lastNivcsw   int64
}

func NewRusageStats(client stats.Client, tags ...stats.Tag) *RusageStats {
	return &RusageStats{
		Utime:    client.Counter("rusage.utime.seconds", tags...),
		Stime:    client.Counter("rusage.stime.seconds", tags...),
		Maxrss:   client.Gauge("rusage.maxrss.bytes", tags...),
		Ixrss:    client.Gauge("rusage.ixrss.bytes", tags...),
		Idrss:    client.Gauge("rusage.idrss.bytes", tags...),
		Isrss:    client.Gauge("rusage.isrss.bytes", tags...),
		Minflt:   client.Counter("rusage.minftl.count", tags...),
		Majflt:   client.Counter("rusage.majflt.count", tags...),
		Nswap:    client.Counter("rusage.nswap.count", tags...),
		Inblock:  client.Counter("rusage.inblock.count", tags...),
		Oublock:  client.Counter("rusage.oublock.count", tags...),
		Msgsnd:   client.Counter("rusage.msgsnd.count", tags...),
		Msgrcv:   client.Counter("rusage.msgrcv.count", tags...),
		Nsignals: client.Counter("rusage.nsignals.count", tags...),
		Nvcsw:    client.Counter("rusage.nvcsw.count", tags...),
		Nivcsw:   client.Counter("rusage.nivcsw.count", tags...),
	}
}

func (ru *RusageStats) Collect() {
	rusage := &syscall.Rusage{}
	syscall.Getrusage(syscall.RUSAGE_SELF, rusage)

	utime := timevalToDuration(rusage.Utime)
	stime := timevalToDuration(rusage.Stime)

	ru.Utime.Add((utime - ru.lastUtime).Seconds())
	ru.Stime.Add((stime - ru.lastStime).Seconds())
	ru.Maxrss.Set(float64(rusage.Maxrss))
	ru.Ixrss.Set(float64(rusage.Ixrss))
	ru.Idrss.Set(float64(rusage.Idrss))
	ru.Isrss.Set(float64(rusage.Isrss))
	ru.Minflt.Add(float64(rusage.Minflt - ru.lastMinflt))
	ru.Majflt.Add(float64(rusage.Majflt - ru.lastMajflt))
	ru.Nswap.Add(float64(rusage.Nswap - ru.lastNswap))
	ru.Inblock.Add(float64(rusage.Inblock - ru.lastInblock))
	ru.Oublock.Add(float64(rusage.Oublock - ru.lastOublock))
	ru.Msgsnd.Add(float64(rusage.Msgsnd - ru.lastMsgsnd))
	ru.Msgrcv.Add(float64(rusage.Msgrcv - ru.lastMsgrcv))
	ru.Nsignals.Add(float64(rusage.Nsignals - ru.lastNsignals))
	ru.Nvcsw.Add(float64(rusage.Nvcsw - ru.lastNvcsw))
	ru.Nivcsw.Add(float64(rusage.Nivcsw - ru.lastNivcsw))

	ru.lastUtime = utime
	ru.lastStime = stime
	ru.lastMinflt = rusage.Minflt
	ru.lastMajflt = rusage.Majflt
	ru.lastNswap = rusage.Nswap
	ru.lastInblock = rusage.Inblock
	ru.lastOublock = rusage.Oublock
	ru.lastMsgsnd = rusage.Msgsnd
	ru.lastMsgrcv = rusage.Msgrcv
	ru.lastNsignals = rusage.Nsignals
	ru.lastNvcsw = rusage.Nvcsw
	ru.lastNivcsw = rusage.Nivcsw
}

func timevalToDuration(t syscall.Timeval) time.Duration {
	return (time.Duration(t.Sec) * time.Second) + (time.Duration(t.Usec) * time.Microsecond)
}
