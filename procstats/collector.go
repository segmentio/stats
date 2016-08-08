package procstats

import (
	"time"

	"github.com/segmentio/stats"
)

type Collector interface {
	Stop()
}

type CollectorConfig struct {
	Client          stats.Client
	CollectInterval time.Duration
}

func NewCollector(client stats.Client) Collector {
	return NewCollectorWith(CollectorConfig{Client: client})
}

func NewCollectorWith(config CollectorConfig) Collector {
	config = setCollectorConfigDefault(config)

	collec := &collector{
		rusage:  NewRusageStats(config.Client),
		runtime: NewRuntimeStats(config.Client),
		memory:  NewMemoryStats(config.Client),
		tick:    time.NewTicker(config.CollectInterval),
		done:    make(chan struct{}),
		join:    make(chan struct{}),
	}

	go collec.run()
	return collec
}

func setCollectorConfigDefault(config CollectorConfig) CollectorConfig {
	if config.CollectInterval == 0 {
		config.CollectInterval = 5 * time.Second
	}

	return config
}

type collector struct {
	rusage  *RusageStats
	runtime *RuntimeStats
	memory  *MemoryStats

	tick *time.Ticker
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
	c.rusage.Collect()
	c.runtime.Collect()
	c.memory.Collect()
}
