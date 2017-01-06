package main

import (
	"flag"
	"log"
	"net"

	"github.com/segmentio/stats/datadog"
)

func main() {
	var bind string

	flag.StringVar(&bind, "bind", ":8125", "The network address to listen on for incoming UDP datagrams")
	flag.Parse()
	log.Printf("listening for incoming UDP datagram on %s", bind)

	datadog.ListenAndServe(bind, datadog.HandlerFunc(func(metric datadog.Metric, from net.Addr) {
		log.Print(metric)
	}))
}
