package procstats

import (
	"os"
	"strconv"
	"time"

	"github.com/segmentio/stats"
)

type Collector interface {
	Stop()
}

type CollectorConfig struct {
	Client          stats.Client
	Pid             int
	CollectInterval time.Duration
}

func NewCollector(client stats.Client) Collector {
	return NewCollectorWith(CollectorConfig{Client: client})
}

func NewCollectorWith(config CollectorConfig) Collector {
	config = setCollectorConfigDefault(config)

	collec := &collector{
		tick: time.NewTicker(config.CollectInterval),
		pid:  config.Pid,
		self: config.Pid == os.Getpid(),
		done: make(chan struct{}),
		join: make(chan struct{}),
	}

	tags := stats.Tags{
		{"pid", strconv.Itoa(collec.pid)},
	}

	if collec.self {
		collec.runtime = NewRuntimeStats(config.Client, tags...)
		collec.memory = NewMemoryStats(config.Client, tags...)
	}

	go collec.run()
	return collec
}

func setCollectorConfigDefault(config CollectorConfig) CollectorConfig {
	if config.Pid == 0 {
		config.Pid = os.Getpid()
	}

	if config.CollectInterval == 0 {
		config.CollectInterval = 5 * time.Second
	}

	return config
}

type collector struct {
	runtime *RuntimeStats
	memory  *MemoryStats

	tick *time.Ticker
	pid  int
	self bool
	done chan struct{}
	join chan struct{}
}

func (c *collector) Stop() {
	c.stop()
	c.wait()
}

func (c *collector) stop() {
	defer func() { recover() }()
	close(c.done)
}

func (c *collector) wait() {
	<-c.join
}

func (c *collector) run() {
	defer close(c.join)
	for {
		select {
		case <-c.tick.C:
			c.collect()

		case <-c.done:
			return
		}
	}
}

func (c *collector) collect() {
	if c.self {
		c.runtime.Collect()
		c.memory.Collect()
	}
}
