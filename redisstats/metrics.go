package redisstats

import "time"

type metrics struct {
	req requestMetrics
	cmd []commandMetrics
}

type requestMetrics struct {
	request struct {
		count int `metric:"count" type:"counter"`
		size  int `metric:"size"  type:"histogram"`
	} `metric:"redis.request"`
}

type commandMetrics struct {
	command struct {
		count int           `metric:"count"       type:"counter"`
		size  int           `metric:"size"        type:"histogram"`
		rsize int           `metric:"return.size" type:"histogram"`
		rtt   time.Duration `metric:"rtt.seconds" type:"histogram"`
	} `metric:"redis.command"`

	error struct {
		count     int    `metric:"count" type:"counter"`
		operation string `tag:"operation'`
		errtype   string `tag:"type"`
	} `metric:"redis.error"`

	name string `tag:"command"`
}
