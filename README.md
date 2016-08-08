# stats [![CircleCI](https://circleci.com/gh/segmentio/stats.svg?style=shield)](https://circleci.com/gh/segmentio/stats) [![GoDoc](https://godoc.org/github.com/segmentio/stats?status.svg)](https://godoc.org/github.com/segmentio/stats)

A Go package for abstracting stats collection.

Installation
------------

```
go get github.com/segmentio/stats
```

Quick Start
-----------

### Backends

The package's design allow for plugging one or more backends to the high-level
`Client` interface. It makes it possible to send metrics to different locations,
or easily change where to send metrics without having to make changes to the code.

Here's an example of how to create a stats client with multiple backends:
```go
package main

import (
    "log"
    "os"

    "github.com/segmentio/stats"
    "github.com/segmentio/stats/datadog"
    "github.com/segmentio/stats/logstats/log"
)

func main() {
    // Create a stats client that sends data to a datadog agent and logs the events.
    client := stats.NewClient("app", stats.MultiBackend(
        datadog.NewBackend("localhost:8125"),
        logstats.NewBackend(log.New(os.Stderr, "stats: ", log.Lstdflags)),
    ))
    defer client.Close()

    // ...

}
```

### Metrics

The `Client` interface makes it easy to declare metrics, common metric types are supported:

- [Gauges](https://godoc.org/github.com/segmentio/stats#Gauge)
- [Counters](https://godoc.org/github.com/segmentio/stats#Counter)
- [Histograms](https://godoc.org/github.com/segmentio/stats#Histogram)

```go
package main

import (
    "github.com/segmentio/stats"
    "github.com/segmentio/stats/datadog"
)

func main() {
    client := stats.NewClient("app", datadog.NewBackend("localhost:8125"))
    defer client.Close()

    // Define a couple of metrics.
    userLogin := client.Counter("users.login")
    userLogout := client.Counter("users.logout")

    // Bump the counters.
    userLogin.Add(1)
    defer userLogout.Add(1)

    // We can add some tags to the metrics as well.
    userLogin.Add(1, stats.Tag{"user", "luke"})

    // ...
}
```

Monitoring
----------

### Processes

The [github.com/segmentio/stats/httpstats](https://godoc.org/github.com/segmentio/stats/procstats)
exposes an API for creating stats collector on local processes. Stats are collected for current
the process and metrics like goroutines count or memory usage are reported.

Here's an example of how to use the collector:
```go
package main

import (
    "github.com/segmentio/stats"
    "github.com/segmentio/stats/procstats"
)


func main() {
    client := stats.NewClient("app", datadog.NewBackend("localhost:8125"))
    defer client.Close()

    // Creates a new stats collector for the current process.
    collector := procstats.NewCollector(client)

    // Gracefully stops stats collection.
    defer collector.Stop()

    // ...
}
```

### HTTP Servers

The [github.com/segmentio/stats/httpstats](https://godoc.org/github.com/segmentio/stats/httpstats)
package exposes a decorator of `http.Handler` that automatically adds metric
colleciton to a HTTP handler, reporting things like request processing time,
error counters, header and body sizes...

Here's an example of how to use the decorator:
```go
package main

import (
    "net/http"

    "github.com/segmentio/stats"
    "github.com/segmentio/stats/httpstats"
)

func main() {
    client := stats.NewClient("app", datadog.NewBackend("localhost:8125"))
    defer client.Close()

    // ...

    http.ListenAndServe(":8080", httpstats.NewHandler(client,
        http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
            // This HTTP handler is automatically reporting metrics for all
            // requests it handles.
            // ...
        }),
    ))
}
```

### HTTP Clients

The [github.com/segmentio/stats/httpstats](https://godoc.org/github.com/segmentio/stats/httpstats)
package exposes a decorator of `http.RoundTripper` which collects and reports
metrics for client requests the same way it's done on the server side.

Here's an exmaple of how to use the decorator:
```go
package main

import (
    "net/http"

    "github.com/segmentio/stats"
    "github.com/segmentio/stats/httpstats"
)

func main() {
    client := stats.NewClient("app", datadog.NewBackend("localhost:8125"))
    defer client.Close()

    // Make a new HTTP client with a transport that will report HTTP metrics.
    httpc := &http.Client{
        Transport: httpstats.NewTransport(client, &http.Transport{}),
    }

    // ...
}
```

You can also modify the default HTTP client to automatically get metrics for all
packages using it, this is very convinient to get insights into dependencies.
```go
package main

import (
    "net/http"

    "github.com/segmentio/stats"
    "github.com/segmentio/stats/httpstats"
)

func main() {
    client := stats.NewClient("app", datadog.NewBackend("localhost:8125"))
    defer client.Close()

    // Wraps the default HTTP client's transport.
    http.DefaultClient.Transport = httpstats.NewTransport(client, http.DefaultClient.Transport)

    // ...
}
```
