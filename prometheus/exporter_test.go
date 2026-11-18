package prometheus

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/stats/v5"
)

func TestStartExporter(t *testing.T) {
	engine := stats.NewEngine("svc", stats.Discard, stats.T("env", "test"))

	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", AppName: "my-app", Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	engine.Add("requests", 3)
	engine.Flush()

	res, err := http.Get("http://" + e.Addr().String() + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	for _, want := range []string{"svc_requests", `appName="my-app"`, `env="test"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
	}
}

func TestStartExporterAppNameTag(t *testing.T) {
	engine := stats.NewEngine("svc", stats.Discard)

	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", AppName: "x", AppNameTag: "service", Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	if got := engine.Tags; len(got) != 1 || got[0] != stats.T("service", "x") {
		t.Errorf("tags = %v", got)
	}
}

func TestStartExporterKeepsExistingHandler(t *testing.T) {
	var seen int
	engine := stats.NewEngine("svc", stats.HandlerFunc(func(time.Time, ...stats.Measure) { seen++ }))

	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: engine})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	engine.Incr("n")
	engine.Flush()
	if seen == 0 {
		t.Error("existing handler no longer receives measures")
	}
}

func TestStartExporterErrors(t *testing.T) {
	if _, err := StartExporter(ExporterConfig{}); err == nil {
		t.Error("missing Addr: want error")
	}

	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("a", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	if _, err := StartExporter(ExporterConfig{Addr: e.Addr().String(), Engine: stats.NewEngine("b", stats.Discard)}); err == nil {
		t.Error("address in use: want error")
	}
}

func TestExporterShutdownTwice(t *testing.T) {
	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("a", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := e.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown %d: %v", i, err)
		}
	}
}

func scrapeBody(t *testing.T, e *Exporter) string {
	t.Helper()

	res, err := http.Get("http://" + e.Addr().String() + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestStartExporterUsesCallerHandler(t *testing.T) {
	engine := stats.NewEngine("svc", stats.Discard)
	h := &Handler{MetricTimeout: time.Hour}

	e, err := StartExporter(ExporterConfig{
		Addr:          "127.0.0.1:0",
		Engine:        engine,
		Handler:       h,
		MetricTimeout: time.Millisecond, // ignored when Handler is set
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	if e.Handler != h {
		t.Fatal("Exporter.Handler is not the caller's handler")
	}
	if h.MetricTimeout != time.Hour {
		t.Errorf("MetricTimeout = %v, the caller's handler was modified", h.MetricTimeout)
	}

	engine.Add("hits", 1)
	engine.Flush()
	if body := scrapeBody(t, e); !strings.Contains(body, "svc_hits") {
		t.Errorf("measure missing from the caller's handler output:\n%s", body)
	}
}

func TestStartExporterMetricTimeout(t *testing.T) {
	e, err := StartExporter(ExporterConfig{
		Addr:          "127.0.0.1:0",
		Engine:        stats.NewEngine("svc", stats.Discard),
		MetricTimeout: 7 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	if e.Handler.MetricTimeout != 7*time.Minute {
		t.Errorf("MetricTimeout = %v", e.Handler.MetricTimeout)
	}
}

func TestStartExporterDefaultsAndTimeouts(t *testing.T) {
	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("svc", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	if e.server.ReadHeaderTimeout != 10*time.Second || e.server.WriteTimeout != 30*time.Second || e.server.IdleTimeout != 60*time.Second {
		t.Errorf("default timeouts = %v, %v, %v", e.server.ReadHeaderTimeout, e.server.WriteTimeout, e.server.IdleTimeout)
	}

	custom, err := StartExporter(ExporterConfig{
		Addr:              "127.0.0.1:0",
		Engine:            stats.NewEngine("svc2", stats.Discard),
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer custom.Shutdown(context.Background())

	if custom.server.ReadHeaderTimeout != time.Second || custom.server.WriteTimeout != 2*time.Second || custom.server.IdleTimeout != 3*time.Second {
		t.Errorf("custom timeouts = %v, %v, %v", custom.server.ReadHeaderTimeout, custom.server.WriteTimeout, custom.server.IdleTimeout)
	}
}

func TestStartExporterDefaultEngine(t *testing.T) {
	saved := stats.DefaultEngine.Tags
	t.Cleanup(func() { stats.DefaultEngine.Tags = saved })

	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", AppName: "my-app"})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Shutdown(context.Background())

	for _, tag := range stats.DefaultEngine.Tags {
		if tag == stats.T(DefaultAppNameTag, "my-app") {
			return
		}
	}
	t.Errorf("default engine tags = %v, want %s=my-app", stats.DefaultEngine.Tags, DefaultAppNameTag)
}

func TestStartExporterIsIdempotentOnTheTag(t *testing.T) {
	engine := stats.NewEngine("svc", stats.Discard)
	for i := 0; i < 2; i++ {
		e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", AppName: "x", Engine: engine})
		if err != nil {
			t.Fatal(err)
		}
		defer e.Shutdown(context.Background())
	}
	if len(engine.Tags) != 1 {
		t.Errorf("tags = %v, want one appName tag", engine.Tags)
	}
}

func TestExporterDoneAndErr(t *testing.T) {
	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("svc", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-e.Done():
		t.Fatal("Done is closed while the server is running")
	default:
	}
	if err := e.Err(); err != nil {
		t.Fatalf("Err = %v while running", err)
	}

	if err := e.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-e.Done():
	case <-time.After(time.Second):
		t.Fatal("Done not closed after Shutdown")
	}
	if err := e.Err(); err != nil {
		t.Errorf("Err = %v after a clean shutdown", err)
	}
}

func TestExporterReportsServeFailure(t *testing.T) {
	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("svc", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}
	// Closing the listener out from under the server makes Serve return a non-ErrServerClosed error.
	if err := e.listener.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-e.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Done not closed after the listener failed")
	}
	if e.Err() == nil {
		t.Error("Err is nil after the listener failed")
	}
	if err := e.Shutdown(context.Background()); err == nil {
		t.Error("Shutdown did not report the serve failure")
	}
}

func TestExporterShutdownClosesConnectionsOnTimeout(t *testing.T) {
	e, err := StartExporter(ExporterConfig{Addr: "127.0.0.1:0", Engine: stats.NewEngine("svc", stats.Discard)})
	if err != nil {
		t.Fatal(err)
	}

	// Hold a request open: send the headers of a request but never finish it, so it stays active.
	conn, err := net.Dial("tcp", e.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /metrics HTTP/1.1\r\nHost: x\r\n")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := e.Shutdown(ctx); err == nil {
		t.Fatal("Shutdown with an in-flight request and a short deadline returned nil")
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil || os.IsTimeout(err) {
		t.Errorf("connection still open after Shutdown timed out: %v", err)
	}
}
