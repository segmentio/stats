package datadog

import (
	"bytes"
	"errors"
	"io"
	"net"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"
)

// Handler defines the interface that types must satisfy to process metrics
// received by a dogstatsd server.
type Handler interface {
	// HandleMetric is called when a dogstatsd server receives a metric.
	// The method receives the metric and the address from which it was sent.
	HandleMetric(Metric, net.Addr)

	// HandleEvent is called when a dogstatsd server receives an event.
	// The method receives the metric and the address from which it was sent.
	HandleEvent(Event, net.Addr)
}

// HandlerFunc makes it possible for function types to be used as metric
// handlers on dogstatsd servers.
type HandlerFunc func(Metric, net.Addr)

// HandleMetric calls f(m, a).
func (f HandlerFunc) HandleMetric(m Metric, a net.Addr) {
	f(m, a)
}

// HandleEvent is a no-op for backwards compatibility.
func (f HandlerFunc) HandleEvent(Event, net.Addr) {}

// ListenAndServe starts a new dogstatsd server, listening for UDP datagrams on
// addr and forwarding the metrics to handler.
func ListenAndServe(addr string, handler Handler) (err error) {
	var conn net.PacketConn

	if conn, err = net.ListenPacket("udp", addr); err != nil {
		return err
	}

	err = Serve(conn, handler)
	return err
}

// Serve runs a dogstatsd server, listening for datagrams on conn and forwarding
// the metrics to handler.
func Serve(conn net.PacketConn, handler Handler) error {
	defer conn.Close()

	concurrency := runtime.GOMAXPROCS(-1)
	if concurrency <= 0 {
		concurrency = 1
	}

	err := conn.SetDeadline(time.Time{})
	if err != nil {
		return err
	}

	var errgrp errgroup.Group

	for i := 0; i < concurrency; i++ {
		errgrp.Go(func() error {
			return serve(conn, handler)
		})
	}

	err = errgrp.Wait()
	switch {
	default:
		return err
	case err == nil:
	case errors.Is(err, io.EOF):
	case errors.Is(err, io.ErrClosedPipe):
	case errors.Is(err, io.ErrUnexpectedEOF):
	}

	return nil
}

func serve(conn net.PacketConn, handler Handler) error {
	b := make([]byte, 65536)

	for {
		n, a, err := conn.ReadFrom(b)
		if err != nil {
			return err
		}

		for s := b[:n]; len(s) != 0; {
			off := bytes.IndexByte(s, '\n')
			if off < 0 {
				off = len(s)
			} else {
				off++
			}

			ln := s[:off]
			s = s[off:]

			if bytes.HasPrefix(ln, []byte("_e")) {
				e, err := parseEvent(string(ln))
				if err != nil {
					continue
				}

				handler.HandleEvent(e, a)
				continue
			}

			m, err := parseMetric(string(ln))
			if err != nil {
				continue
			}

			handler.HandleMetric(m, a)
		}
	}
}
