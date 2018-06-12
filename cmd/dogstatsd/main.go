package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	var fset = flag.NewFlagSet("dogstatsd "+cmd+" [options...] metric value [-- args...]", flag.ExitOnError)
	var extra []string
	var tags tags
	var addr string
	var name string
	var value float64
	var err error

	args, extra = split(args, "--")
	fset.StringVar(&addr, "addr", "localhost:8125", "The network address where a dogstatsd server is listening for incoming UDP datagrams")
	fset.Var(&tags, "tags", "A comma-separated list of tags to set on the metric")
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
		} else {
			args = args[1:]
		}

	case "set":
		if len(args) == 0 {
			errorf("missing metric value")
		} else if value, err = strconv.ParseFloat(args[0], 64); err != nil {
			errorf("bad metric value: %s", args[0])
		} else {
			args = args[1:]
		}
	}

	dd := datadog.NewClient(addr)
	defer dd.Close()

	switch cmd {
	case "add":
		stats.Add(name, value, tags...)

	case "set":
		stats.Set(name, value, tags...)

	case "time":
		start := time.Now()
		run(extra...)
		stats.Observe(name, time.Now().Sub(start), tags...)
	}
}

func server(args ...string) {
	var fset = flag.NewFlagSet("dogstatsd agent [options...]", flag.ExitOnError)
	var bind string

	fset.StringVar(&bind, "bind", ":8125", "The network address to listen on for incoming UDP datagrams")
	fset.Parse(args)
	log.Printf("listening for incoming UDP datagram on %s", bind)

	datadog.ListenAndServe(bind, handlers{})
}

type handlers struct{}

func (h handlers) HandleMetric(m datadog.Metric, a net.Addr) {
	log.Print(m)
}

func (h handlers) HandleEvent(e datadog.Event, a net.Addr) {
	log.Print(e)
}

func run(args ...string) {
	if len(args) == 0 {
		errorf("missing command line")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		errorf("%s", err)
	}
}

func errorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func split(args []string, sep string) (head []string, tail []string) {
	if i := indexOf(args, sep); i < 0 {
		head = args
	} else {
		head, tail = args[:i], args[i+1:]
	}
	return
}

func indexOf(args []string, s string) int {
	for i, a := range args {
		if a == s {
			return i
		}
	}
	return -1
}

type tags []stats.Tag

func (tags tags) String() string {
	b := &bytes.Buffer{}

	for i, tag := range tags {
		if i != 0 {
			b.WriteByte(',')
		}
		b.WriteString(tag.Name)
		b.WriteByte(':')
		b.WriteString(tag.Value)
	}

	return b.String()
}

func (tags *tags) Set(s string) (err error) {
	for _, pair := range strings.Split(s, ",") {
		var tag stats.Tag
		if i := strings.IndexByte(pair, ':'); i < 0 {
			tag.Name = pair
		} else {
			tag.Name, tag.Value = pair[:i], pair[i+1:]
		}
		*tags = append(*tags, tag)
	}
	return
}
