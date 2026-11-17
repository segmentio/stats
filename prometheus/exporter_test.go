package prometheus

import (
	"context"
	"io"
	"net/http"
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
