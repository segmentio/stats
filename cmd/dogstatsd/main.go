package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/segmentio/stats"
	"github.com/segmentio/stats/datadog"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		usage()
	}

	switch cmd, args := args[0], args[1:]; cmd {
	case "add", "set", "time":
		client(cmd, args...)
	case "agent":
		server(args...)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: dogstatsd [command] [arguments...]

commands:
 - add
 - agent
 - help
 - set
 - time

`)
	os.Exit(1)
}

func client(cmd string, args ...string) {
	var fset = flag.NewFlagSet("dogstatsd "+cmd, flag.ExitOnError)
	var addr string
	var name string
	var value float64
	var err error

	fset.StringVar(&addr, "addr", "localhost:8125", "The network address where a dogstatsd server is listening for incoming UDP datagrams")
	fset.Parse(args)
	args = fset.Args()

	if len(args) == 0 {
		errorf("missing metric name")
	}

	name, args = args[0], args[1:]

	switch cmd {
	case "add":
		if len(args) == 0 {
			value = 1.0
		} else if value, err = strconv.ParseFloat(args[0], 64); err != nil {
			errorf("bad metric value: %s", args[0])
		}
		args = args[1:]

	case "set":
		if len(args) == 0 {
			errorf("missing metric value")
		} else if value, err = strconv.ParseFloat(args[0], 64); err != nil {
			errorf("bad metric value: %s", args[0])
		}
		args = args[1:]

	case "time":
		if len(args) == 0 {
			args = []string{"true"}
		}
	}

	dd := datadog.NewClient(datadog.ClientConfig{Address: addr})
	defer dd.Close()

	switch cmd {
	case "add":
		stats.Add(name, value)

	case "set":
		stats.Set(name, value)

	case "time":
		clock := stats.Time(name, time.Now())
		run(args...)
		clock.Stop()
	}
}

func server(args ...string) {
	var fset = flag.NewFlagSet("dogstatsd agent", flag.ExitOnError)
	var bind string

	fset.StringVar(&bind, "bind", ":8125", "The network address to listen on for incoming UDP datagrams")
	fset.Parse(args)
	log.Printf("listening for incoming UDP datagram on %s", bind)

	datadog.ListenAndServe(bind, datadog.HandlerFunc(func(metric datadog.Metric, from net.Addr) {
		log.Print(metric)
	}))
}

func run(args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		errorf("%s", err)
	}
}

func errorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
