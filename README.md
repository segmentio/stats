# stats [![CircleCI](https://circleci.com/gh/segmentio/stats.svg?style=shield)](https://circleci.com/gh/segmentio/stats) [![GoDoc](https://godoc.org/github.com/segmentio/stats?status.svg)](https://godoc.org/github.com/segmentio/stats)

A Go package for abstracting stats collection.

Installation
------------

```
go get github.com/segmentio/stats
```

Quick Start
-----------

**Counters**

```go
package main

import (
    "log"
    "os"

    "github.com/segmentio/stats"
    "github.com/segmentio/stats/datadog"
)

func main() {
    // Create a stats client that sends data to a datadog agent and logs the events.
    client := stats.NewClient("app", stats.MultiBackend(
        datadog.NewBackend("localhost:8125"),
        log_stats.NewBackend(log.New(os.Stderr, "stats: ", log.Lstdflags)),
    ))
    defer client.Close()

    // Define a couple of metrics.
    userLogin := client.Counter("users.login")
    userLogout := client.Counter("users.logout")

    // Bump the counters.
    userLogin.Add(1)
    defer userLogout.Add(1)

    // We can add some tags to the metrics as well.
    userLogin.Add(1, stats.Tag{"user", "luke"})

}
```
