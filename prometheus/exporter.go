package prometheus

import (
	"cmp"
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/segmentio/stats/v5"
)

// DefaultAppNameTag is the tag the exporter adds to the engine to identify the
// service that owns the series. Collectors that derive a service name from the
// metrics read "service", then "appName"; "appName" is the default because many
// services already use "service" for the name of a downstream.
const DefaultAppNameTag = "appName"

// ExporterConfig configures StartExporter.
type ExporterConfig struct {
	// Addr is the listen address of the metrics server, for example ":9102".
	// It is required.
	Addr string

	// AppName is the value of the app name tag added to every series. Empty
	// adds no tag.
	AppName string

	// AppNameTag is the name of the tag carrying AppName. The default is
	// DefaultAppNameTag.
	AppNameTag string

	// Engine is the engine the handler is registered on and the tag is added
	// to. The default is stats.DefaultEngine.
	Engine *stats.Engine

	// Handler serves the metrics. The default is a new Handler using
	// MetricTimeout.
	Handler *Handler

	// MetricTimeout is how long a series is exposed after its last update.
	// It is ignored when Handler is set.
	MetricTimeout time.Duration

	// ReadHeaderTimeout bounds reading request headers. The default is 10s.
	ReadHeaderTimeout time.Duration

	// WriteTimeout bounds writing a response, so a slow reader of /metrics
	// cannot hold a connection open. The default is 30s.
	WriteTimeout time.Duration

	// IdleTimeout bounds how long an idle keep-alive connection is kept. The
	// default is 60s.
	IdleTimeout time.Duration
}

// Exporter is a running metrics server started by StartExporter.
type Exporter struct {
	// Handler is the handler registered on the engine.
	Handler *Handler

	server   *http.Server
	addr     net.Addr
	listener net.Listener
	done     chan struct{}
	err      error // set before done is closed
}

// StartExporter serves the engine's metrics at /metrics on cfg.Addr, in
// addition to any handler already registered on the engine, which keeps running
// unchanged.
//
// It adds the app name tag to the engine's tags and registers the handler on the
// engine. Neither write is synchronized with the engine's readers, so call it
// before the engine is used from other goroutines, and before creating any engine
// derived from cfg.Engine (WithTags, WithPrefix), which would not inherit the tag.
// Calling it again on the same engine is safe: the tag is deduplicated by name.
//
// It returns once the listener is bound, so a bad address is an error here rather
// than a log line later. If the server stops on its own later, Done is closed and
// Err returns the cause.
func StartExporter(cfg ExporterConfig) (*Exporter, error) {
	if cfg.Addr == "" {
		return nil, errors.New("prometheus: ExporterConfig.Addr is required")
	}

	engine := cfg.Engine
	if engine == nil {
		engine = stats.DefaultEngine
	}

	if cfg.AppName != "" {
		tag := cfg.AppNameTag
		if tag == "" {
			tag = DefaultAppNameTag
		}
		engine.Tags = stats.SortTags(append(append([]stats.Tag{}, engine.Tags...), stats.T(tag, cfg.AppName)))
	}

	handler := cfg.Handler
	if handler == nil {
		handler = &Handler{MetricTimeout: cfg.MetricTimeout}
	}

	readHeaderTimeout := cmp.Or(cfg.ReadHeaderTimeout, 10*time.Second)
	writeTimeout := cmp.Or(cfg.WriteTimeout, 30*time.Second)
	idleTimeout := cmp.Or(cfg.IdleTimeout, 60*time.Second)

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	e := &Exporter{
		Handler: handler,
		server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		addr:     ln.Addr(),
		listener: ln,
		done:     make(chan struct{}),
	}

	engine.Register(handler)

	go func() {
		defer close(e.done)
		if err := e.server.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			e.err = err
		}
	}()

	return e, nil
}

// Addr returns the address the server is listening on.
func (e *Exporter) Addr() net.Addr { return e.addr }

// Done is closed when the server has stopped, either through Shutdown or because
// it failed.
func (e *Exporter) Done() <-chan struct{} { return e.done }

// Err returns the error that stopped the server, or nil if it was shut down
// normally or is still running. It is only meaningful once Done is closed.
func (e *Exporter) Err() error {
	select {
	case <-e.done:
		return e.err
	default:
		return nil
	}
}

// Shutdown stops the server, waiting for in-flight scrapes until ctx is done, then
// closing any connection still open. It returns the error that stopped the server
// early, if any. The handler stays registered on the engine.
func (e *Exporter) Shutdown(ctx context.Context) error {
	err := e.server.Shutdown(ctx)
	if err != nil {
		e.server.Close() // ctx expired with scrapes in flight: do not leave them open
	}
	<-e.done
	if err == nil {
		err = e.err
	}
	return err
}
